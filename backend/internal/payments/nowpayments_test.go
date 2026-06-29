package payments

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func sign(secret string, body []byte) string {
	var generic map[string]any
	_ = json.Unmarshal(body, &generic)
	sorted := sortedJSON(generic)
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(sorted)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyIPN(t *testing.T) {
	np := NewNowPayments("", "ipn-secret", "https://api.nowpayments.io", "", "", "")
	body := []byte(`{"order_id":"pay_123","payment_status":"finished","price_amount":10.5,"actually_paid":10.5,"pay_currency":"btc"}`)
	sig := sign("ipn-secret", body)

	p, err := np.VerifyIPN(body, sig)
	if err != nil {
		t.Fatalf("expected valid signature, got %v", err)
	}
	if p.OrderID != "pay_123" || p.PaymentStatus != "finished" {
		t.Fatalf("unexpected payload: %+v", p)
	}
}

func TestVerifyIPNRejectsBadSignature(t *testing.T) {
	np := NewNowPayments("", "ipn-secret", "https://api.nowpayments.io", "", "", "")
	body := []byte(`{"order_id":"pay_123","payment_status":"finished"}`)
	if _, err := np.VerifyIPN(body, "deadbeef"); err == nil {
		t.Fatal("expected bad signature to be rejected")
	}
}

func TestVerifyIPNRejectsTamperedBody(t *testing.T) {
	np := NewNowPayments("", "ipn-secret", "https://api.nowpayments.io", "", "", "")
	body := []byte(`{"order_id":"pay_123","payment_status":"finished","actually_paid":10}`)
	sig := sign("ipn-secret", body)
	tampered := []byte(`{"order_id":"pay_123","payment_status":"finished","actually_paid":9999}`)
	if _, err := np.VerifyIPN(tampered, sig); err == nil {
		t.Fatal("expected tampered body to fail verification")
	}
}
