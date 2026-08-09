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

// The signatures below were produced by NOWPayments' documented Node signing
// scheme — crypto.createHmac('sha512', secret).update(JSON.stringify(payload,
// Object.keys(payload).sort())) — rather than by this package. They are an
// independent oracle: sign() above mirrors our own canonicalisation and would
// happily agree with a wrong implementation, these vectors would not.
func TestVerifyIPNMatchesJavaScriptCanonicalisation(t *testing.T) {
	np := NewNowPayments("", "ipn-secret", "https://api.nowpayments.io", "", "", "")
	cases := []struct {
		name, body, sig string
	}{
		{
			name: "plain",
			body: `{"order_id":"pay_123","payment_status":"finished","price_amount":10.5,"actually_paid":10.5,"pay_currency":"btc"}`,
			sig:  "6bf7b2a291d8c103d27230d86ba627cc17794b669e3fcfff433e1a0e0d92e08c6cc70232226132d232b5047270df27fafd15777e463b95def4871d9296f0fd42",
		},
		{
			// JSON.stringify does not escape <, > or &; encoding/json does by
			// default, which would break verification for these payloads.
			name: "html characters",
			body: `{"order_id":"pay_9","payment_status":"finished","order_description":"Top-up <A&B>","outcome_amount":0.00001234}`,
			sig:  "a467408d2480ce7a9b88464df804ca51817ae485734851490e4d3bdbfb5caa2df38829f790f90888ce48b71ec90fc8f35a4c891b29e42b77fea65466b99f087f",
		},
		{
			// JSON.stringify renders 0.000000123 as 1.23e-7 and 1000000 as
			// 1000000; echoing the wire literals instead would not verify.
			name: "exponential and large numbers",
			body: `{"order_id":"pay_7","payment_id":5745459419,"price_amount":1000000,"actually_paid":0.000000123}`,
			sig:  "9c2ce8b16898bf0590def23c845d4e4eb5288ddd7f1df7589a1b8168226b3a2e2e8729923ba329a73cb37c0fbda552c5f66c920d4c03164e4fd524b4213bb6b3",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := np.VerifyIPN([]byte(tc.body), tc.sig); err != nil {
				t.Fatalf("signature from the reference implementation was rejected: %v", err)
			}
		})
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
