package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
)

func (a *App) handleWallet(w http.ResponseWriter, r *http.Request) {
	uid, _ := userID(r)
	u, err := a.Store.GetUserByID(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"balance_cents": u.BalanceCents, "currency": "usd"})
}

func (a *App) handleLedger(w http.ResponseWriter, r *http.Request) {
	uid, _ := userID(r)
	entries, err := a.Store.ListLedger(r.Context(), uid, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load ledger")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

func (a *App) handleListPayments(w http.ResponseWriter, r *http.Request) {
	uid, _ := userID(r)
	payments, err := a.Store.ListPayments(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load payments")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payments": payments})
}

type topupReq struct {
	AmountCents int64  `json:"amount_cents" validate:"required,min=1"`
	Provider    string `json:"provider" validate:"required,oneof=stripe nowpayments"`
}

func (a *App) handleTopup(w http.ResponseWriter, r *http.Request) {
	uid, _ := userID(r)
	var req topupReq
	if !decode(w, r, &req) {
		return
	}
	if req.AmountCents < a.Cfg.MinTopupCents {
		writeError(w, http.StatusBadRequest, "amount below minimum top-up")
		return
	}
	if req.AmountCents > 1_000_000_00 { // hard ceiling: $1,000,000
		writeError(w, http.StatusBadRequest, "amount too large")
		return
	}
	u, err := a.Store.GetUserByID(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start payment")
		return
	}
	paymentID := uuid.NewString()

	switch req.Provider {
	case "stripe":
		if !a.Stripe.Enabled() {
			writeError(w, http.StatusNotImplemented, "card payments are not configured")
			return
		}
		url, sessionID, err := a.Stripe.CreateCheckoutSession(uid.String(), paymentID, u.Email, req.AmountCents)
		if err != nil {
			a.Log.Error("stripe checkout", "err", err)
			writeError(w, http.StatusBadGateway, "could not start card payment")
			return
		}
		if _, err := a.Store.CreatePayment(r.Context(), uid, "stripe", sessionID, req.AmountCents, "usd",
			map[string]any{"payment_id": paymentID}); err != nil {
			a.Log.Error("create payment", "err", err)
			writeError(w, http.StatusInternalServerError, "could not record payment")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"checkout_url": url})

	case "nowpayments":
		if !a.Now.Enabled() {
			writeError(w, http.StatusNotImplemented, "crypto payments are not configured")
			return
		}
		if _, err := a.Store.CreatePayment(r.Context(), uid, "nowpayments", paymentID, req.AmountCents, "usd", nil); err != nil {
			a.Log.Error("create payment", "err", err)
			writeError(w, http.StatusInternalServerError, "could not record payment")
			return
		}
		url, _, err := a.Now.CreateInvoice(r.Context(), paymentID, req.AmountCents)
		if err != nil {
			a.Log.Error("nowpayments invoice", "err", err)
			writeError(w, http.StatusBadGateway, "could not start crypto payment")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"checkout_url": url})
	}
}

// ---- Webhooks ----

func (a *App) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read body")
		return
	}
	event, err := a.Stripe.ConstructEvent(payload, r.Header.Get("Stripe-Signature"))
	if err != nil {
		a.Log.Warn("stripe signature verification failed", "err", err)
		writeError(w, http.StatusBadRequest, "invalid signature")
		return
	}
	// Idempotency on the Stripe event id.
	if first, err := a.Store.MarkWebhookSeen(r.Context(), "stripe", event.ID, string(event.Type)); err != nil {
		writeError(w, http.StatusInternalServerError, "error")
		return
	} else if !first {
		writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var sess struct {
			ID            string `json:"id"`
			PaymentStatus string `json:"payment_status"`
		}
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			writeError(w, http.StatusBadRequest, "bad payload")
			return
		}
		if sess.PaymentStatus == "paid" {
			if _, err := a.Store.MarkPaymentPaidAndCredit(r.Context(), "stripe", sess.ID); err != nil {
				a.Log.Error("credit stripe payment", "err", err, "session", sess.ID)
				writeError(w, http.StatusInternalServerError, "could not credit")
				return
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) handleNowPaymentsWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read body")
		return
	}
	ipn, err := a.Now.VerifyIPN(body, r.Header.Get("x-nowpayments-sig"))
	if err != nil {
		a.Log.Warn("nowpayments IPN verification failed", "err", err)
		writeError(w, http.StatusBadRequest, "invalid signature")
		return
	}
	ref := ipn.OrderID
	if ref == "" {
		writeError(w, http.StatusBadRequest, "missing order id")
		return
	}
	// Idempotency keyed on order id + status transition.
	eventKey := ref + ":" + ipn.PaymentStatus
	if first, _ := a.Store.MarkWebhookSeen(r.Context(), "nowpayments", eventKey, ipn.PaymentStatus); !first {
		writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
		return
	}

	switch ipn.PaymentStatus {
	case "finished", "confirmed", "partially_paid":
		// Credit only fully-paid invoices.
		if ipn.PaymentStatus == "partially_paid" {
			_ = a.Store.UpdatePaymentStatus(r.Context(), "nowpayments", ref, "confirming")
			break
		}
		if _, err := a.Store.MarkPaymentPaidAndCredit(r.Context(), "nowpayments", ref); err != nil {
			a.Log.Error("credit crypto payment", "err", err, "ref", ref)
			writeError(w, http.StatusInternalServerError, "could not credit")
			return
		}
	case "failed", "expired", "refunded":
		_ = a.Store.UpdatePaymentStatus(r.Context(), "nowpayments", ref, ipn.PaymentStatus)
	default:
		_ = a.Store.UpdatePaymentStatus(r.Context(), "nowpayments", ref, "confirming")
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
