// Package catalog syncs the VaultProxies reseller product catalog into the
// local plans table, applying the configured reseller markup to derive the
// retail price customers pay. This keeps the storefront's pricing correct and
// in lock-step with the upstream — no hardcoded prices.
package catalog

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/vaultproxies/vpx/backend/internal/store"
	"github.com/vaultproxies/vpx/backend/internal/vaultproxies"
)

// typeOrder controls how proxy types are grouped in the storefront.
var typeOrder = map[string]int{
	"residential": 10,
	"isp":         30,
	"datacenter":  40,
	"ipv6":        60,
	"mobile":      70,
}

type Result struct {
	Synced      int      `json:"synced"`
	Deactivated int64    `json:"deactivated"`
	Codes       []string `json:"codes"`
}

// Sync fetches the upstream catalog and upserts it into the plans table with
// retail = round(wholesale * markup). Idempotent; safe to run on a schedule.
func Sync(ctx context.Context, st *store.Store, client vaultproxies.Client, markup float64, log *slog.Logger) (*Result, error) {
	if markup < 1.0 {
		markup = 1.0
	}
	products, err := client.Catalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch upstream catalog: %w", err)
	}
	if len(products) == 0 {
		return nil, fmt.Errorf("upstream catalog is empty")
	}

	// Stable ordering: by proxy type, then wholesale price.
	sort.SliceStable(products, func(i, j int) bool {
		if typeOrder[products[i].Type] != typeOrder[products[j].Type] {
			return typeOrder[products[i].Type] < typeOrder[products[j].Type]
		}
		return products[i].WholesaleCents < products[j].WholesaleCents
	})

	codes := make([]string, 0, len(products))
	for i, p := range products {
		retail := int64(float64(p.WholesaleCents)*markup + 0.5)
		if retail < p.WholesaleCents {
			retail = p.WholesaleCents
		}
		sortOrder := typeOrder[p.Type] + i
		name := p.Name
		if name == "" {
			name = p.Code
		}
		if err := st.UpsertPlan(ctx, p.Code, name, p.Type, p.Unit, p.WholesaleCents, retail, p.MinQuantity, sortOrder); err != nil {
			return nil, fmt.Errorf("upsert plan %s: %w", p.Code, err)
		}
		codes = append(codes, p.Code)
	}

	deactivated, err := st.DeactivatePlansNotIn(ctx, codes)
	if err != nil {
		return nil, fmt.Errorf("deactivate stale plans: %w", err)
	}

	if log != nil {
		log.Info("catalog synced", "products", len(codes), "deactivated", deactivated, "markup", markup)
	}
	return &Result{Synced: len(codes), Deactivated: deactivated, Codes: codes}, nil
}
