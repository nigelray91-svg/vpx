package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/vaultproxies/vpx/backend/internal/models"
	"github.com/vaultproxies/vpx/backend/internal/store"
	"github.com/vaultproxies/vpx/backend/internal/vaultproxies"
	"github.com/go-chi/chi/v5"
)

type createOrderReq struct {
	PlanID    string `json:"plan_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"required,min=1,max=100000"`
	Rotation  string `json:"rotation" validate:"omitempty,oneof=rotating sticky"`
	StickyTTL int    `json:"sticky_ttl_seconds" validate:"omitempty,min=0,max=3600"`
	Region    string `json:"region" validate:"omitempty,max=32"`
}

func (a *App) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	uid, _ := userID(r)
	var req createOrderReq
	if !decode(w, r, &req) {
		return
	}
	planID, _ := uuid.Parse(req.PlanID)
	plan, err := a.Store.GetPlanByID(r.Context(), planID)
	if err != nil || !plan.Active {
		writeError(w, http.StatusBadRequest, "plan not available")
		return
	}
	if req.Quantity < plan.MinQuantity {
		writeError(w, http.StatusBadRequest, "quantity below plan minimum")
		return
	}
	rotation := req.Rotation
	if rotation == "" {
		rotation = vaultproxies.RotationRotating
	}
	total := plan.RetailCentsUnit * int64(req.Quantity)

	// 1) Create the order row (pending).
	order, err := a.Store.CreateOrder(r.Context(), uid, plan.ID, req.Quantity, plan.Unit, total)
	if err != nil {
		a.Log.Error("create order", "err", err)
		writeError(w, http.StatusInternalServerError, "could not create order")
		return
	}

	// 2) Charge the wallet atomically.
	if _, err := a.Store.Debit(r.Context(), uid, total, order.ID.String(), "Order "+plan.Code); err != nil {
		if errors.Is(err, store.ErrInsufficientFunds) {
			_ = a.Store.UpdateOrderStatus(r.Context(), order.ID, "cancelled", nil, nil)
			writeErrorCode(w, http.StatusPaymentRequired, "insufficient wallet balance", "insufficient_funds")
			return
		}
		a.Log.Error("debit", "err", err)
		writeError(w, http.StatusInternalServerError, "payment failed")
		return
	}

	// 3) Provision upstream.
	_ = a.Store.UpdateOrderStatus(r.Context(), order.ID, "provisioning", nil, nil)
	result, err := a.Vault.Provision(r.Context(), vaultproxies.ProvisionRequest{
		ProxyType:    plan.ProxyType,
		Unit:         plan.Unit,
		Quantity:     req.Quantity,
		Rotation:     rotation,
		StickyTTLSec: req.StickyTTL,
		Pool:         plan.Code,
		Region:       req.Region,
		Label:        order.ID.String(),
	})
	if err != nil {
		// Refund and mark failed — the customer is made whole.
		a.Log.Error("provision failed, refunding", "order", order.ID, "err", err)
		_, _ = a.Store.Credit(r.Context(), uid, "refund", total, order.ID.String(), "Refund: provisioning failed")
		_ = a.Store.UpdateOrderStatus(r.Context(), order.ID, "failed", nil, nil)
		writeErrorCode(w, http.StatusBadGateway, "provisioning failed, you were refunded", "provision_failed")
		return
	}

	// 4) Persist credentials.
	ref := result.Ref
	var proxies []models.Proxy
	for _, c := range result.Proxies {
		p := &models.Proxy{
			UserID:              uid,
			OrderID:             order.ID,
			ProxyType:           plan.ProxyType,
			Protocol:            firstNonEmpty(c.Protocol, "http"),
			Host:                c.Host,
			Port:                c.Port,
			Username:            c.Username,
			Password:            c.Password,
			Pool:                c.Pool,
			Rotation:            firstNonEmpty(c.Rotation, rotation),
			StickyTTLSeconds:    c.StickyTTLSeconds,
			BandwidthLimitBytes: c.BandwidthLimitBytes,
			ExpiresAt:           result.ExpiresAt,
		}
		saved, err := a.Store.CreateProxy(r.Context(), p, ref)
		if err != nil {
			a.Log.Error("save proxy", "err", err)
			continue
		}
		proxies = append(proxies, *saved)
	}
	_ = a.Store.UpdateOrderStatus(r.Context(), order.ID, "active", &ref, result.ExpiresAt)
	order.Status = "active"
	order.VaultRef = &ref

	writeJSON(w, http.StatusCreated, map[string]any{"order": order, "proxies": proxies})
}

func (a *App) handleListOrders(w http.ResponseWriter, r *http.Request) {
	uid, _ := userID(r)
	orders, err := a.Store.ListOrders(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load orders")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (a *App) handleListProxies(w http.ResponseWriter, r *http.Request) {
	uid, _ := userID(r)
	proxies, err := a.Store.ListProxies(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load proxies")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"proxies": proxies})
}

func (a *App) handleProxyUsage(w http.ResponseWriter, r *http.Request) {
	uid, _ := userID(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid proxy id")
		return
	}
	proxy, err := a.Store.GetProxy(r.Context(), id, uid)
	if err != nil {
		writeError(w, http.StatusNotFound, "proxy not found")
		return
	}
	order, err := a.Store.GetOrder(r.Context(), proxy.OrderID, uid)
	if err != nil || order.VaultRef == nil {
		writeJSON(w, http.StatusOK, map[string]any{"bandwidth_used_bytes": proxy.BandwidthUsedBytes})
		return
	}
	usage, err := a.Vault.Usage(r.Context(), *order.VaultRef)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"bandwidth_used_bytes": proxy.BandwidthUsedBytes})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"bandwidth_used_bytes": usage.BandwidthUsedBytes,
		"bandwidth_cap_bytes":  usage.BandwidthCapBytes,
		"active":               usage.Active,
	})
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
