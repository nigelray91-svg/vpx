package payments

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// NowPayments integrates the NOWPayments crypto invoice + IPN flow.
type NowPayments struct {
	apiKey     string
	ipnSecret  string
	baseURL    string
	successURL string
	cancelURL  string
	ipnURL     string
	http       *http.Client
}

func NewNowPayments(apiKey, ipnSecret, baseURL, successURL, cancelURL, ipnURL string) *NowPayments {
	return &NowPayments{
		apiKey:     apiKey,
		ipnSecret:  ipnSecret,
		baseURL:    strings.TrimRight(baseURL, "/"),
		successURL: successURL,
		cancelURL:  cancelURL,
		ipnURL:     ipnURL,
		http:       &http.Client{Timeout: 15 * time.Second},
	}
}

func (n *NowPayments) Enabled() bool { return n.apiKey != "" }

type invoiceReq struct {
	PriceAmount   float64 `json:"price_amount"`
	PriceCurrency string  `json:"price_currency"`
	OrderID       string  `json:"order_id"`
	OrderDesc     string  `json:"order_description"`
	IPNCallback   string  `json:"ipn_callback_url"`
	SuccessURL    string  `json:"success_url"`
	CancelURL     string  `json:"cancel_url"`
}

type invoiceResp struct {
	ID         string `json:"id"`
	InvoiceURL string `json:"invoice_url"`
}

// CreateInvoice creates a hosted crypto invoice for the given USD amount.
// paymentID is our internal payment id, echoed back as order_id in the IPN.
func (n *NowPayments) CreateInvoice(ctx context.Context, paymentID string, amountCents int64) (invoiceURL, providerID string, err error) {
	body := invoiceReq{
		PriceAmount:   float64(amountCents) / 100.0,
		PriceCurrency: "usd",
		OrderID:       paymentID,
		OrderDesc:     "Wallet top-up",
		IPNCallback:   n.ipnURL,
		SuccessURL:    n.successURL,
		CancelURL:     n.cancelURL,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.baseURL+"/v1/invoice", bytes.NewReader(b))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("x-api-key", n.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.http.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("nowpayments invoice: %d %s", resp.StatusCode, string(data))
	}
	var out invoiceResp
	if err := json.Unmarshal(data, &out); err != nil {
		return "", "", err
	}
	if out.InvoiceURL == "" {
		return "", "", fmt.Errorf("nowpayments: empty invoice url")
	}
	return out.InvoiceURL, out.ID, nil
}

// IPNPayload is the subset of the NOWPayments IPN we rely on.
type IPNPayload struct {
	PaymentID       any     `json:"payment_id"`
	PaymentStatus   string  `json:"payment_status"`
	OrderID         string  `json:"order_id"`
	PriceAmount     float64 `json:"price_amount"`
	PriceCurrency   string  `json:"price_currency"`
	ActuallyPaid    float64 `json:"actually_paid"`
	PayCurrency     string  `json:"pay_currency"`
}

// VerifyIPN validates the HMAC-SHA512 signature NOWPayments sends in the
// `x-nowpayments-sig` header. The signature is computed over the JSON body with
// keys sorted alphabetically.
func (n *NowPayments) VerifyIPN(body []byte, signature string) (*IPNPayload, error) {
	if n.ipnSecret == "" {
		return nil, fmt.Errorf("nowpayments IPN secret not configured")
	}
	// Re-serialize with sorted keys to match NOWPayments' signing scheme.
	var generic map[string]any
	if err := json.Unmarshal(body, &generic); err != nil {
		return nil, err
	}
	sorted := sortedJSON(generic)
	mac := hmac.New(sha512.New, []byte(n.ipnSecret))
	mac.Write(sorted)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature))) {
		return nil, fmt.Errorf("invalid IPN signature")
	}
	var p IPNPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// sortedJSON marshals a map with keys sorted recursively, matching the format
// NOWPayments uses to compute the IPN HMAC.
func sortedJSON(v any) []byte {
	var buf bytes.Buffer
	writeSorted(&buf, v)
	return buf.Bytes()
}

func writeSorted(buf *bytes.Buffer, v any) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			writeSorted(buf, t[k])
		}
		buf.WriteByte('}')
	case []any:
		buf.WriteByte('[')
		for i, e := range t {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeSorted(buf, e)
		}
		buf.WriteByte(']')
	default:
		b, _ := json.Marshal(t)
		buf.Write(b)
	}
}
