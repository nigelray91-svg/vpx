// Package payments integrates Stripe (cards) and NOWPayments (crypto) for
// wallet top-ups.
package payments

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/webhook"
)

type Stripe struct {
	secretKey     string
	webhookSecret string
	successURL    string
	cancelURL     string
}

func NewStripe(secretKey, webhookSecret, successURL, cancelURL string) *Stripe {
	stripe.Key = secretKey
	return &Stripe{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		successURL:    successURL,
		cancelURL:     cancelURL,
	}
}

func (s *Stripe) Enabled() bool { return s.secretKey != "" }

// CreateCheckoutSession builds a hosted Checkout page for a wallet top-up.
// The userID and our internal payment id travel in metadata and client_reference_id.
func (s *Stripe) CreateCheckoutSession(userID, paymentID, email string, amountCents int64) (url, sessionID string, err error) {
	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		ClientReferenceID: stripe.String(paymentID),
		CustomerEmail:     stripe.String(email),
		SuccessURL:        stripe.String(s.successURL),
		CancelURL:         stripe.String(s.cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Wallet top-up"),
					},
					UnitAmount: stripe.Int64(amountCents),
				},
				Quantity: stripe.Int64(1),
			},
		},
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{"user_id": userID, "payment_id": paymentID},
		},
		Metadata: map[string]string{"user_id": userID, "payment_id": paymentID},
	}
	params.Context = nil
	sess, err := session.New(params)
	if err != nil {
		return "", "", err
	}
	return sess.URL, sess.ID, nil
}

// ConstructEvent verifies the Stripe webhook signature and returns the event.
func (s *Stripe) ConstructEvent(payload []byte, sigHeader string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, sigHeader, s.webhookSecret)
}
