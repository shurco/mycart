package litepay

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// genRSAPEM generates a PKCS#8-encoded RSA private key in PEM format for
// use in SpectroCoin signing tests.
func genRSAPEM(t *testing.T) string {
	t.Helper()
	// 1024 bits is the smallest RSA size Go will sign with; it's plenty for
	// a test-only key where we're only verifying plumbing, not cryptographic
	// strength.
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("MarshalPKCS8: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func TestSpectrocoin_Pay_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/merchant/1/createOrder" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{
			"receiveAmount": "10.00",
			"receiveCurrency": "EUR",
			"redirectUrl": "https://spectro/pay/1"
		}`))
	}))
	t.Cleanup(srv.Close)
	redirectHTTPClient(t, srv, "https://spectrocoin.com/")

	provider := New("cb", "ok", "cancel").Spectrocoin("merchant-1", "project-1", genRSAPEM(t))
	p, err := provider.Pay(Cart{
		ID:       "SC1234567890123",
		Currency: "EUR",
		Items:    []Item{{PriceData: Price{UnitAmount: 1000}, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if p.Status != PROCESSED || p.AmountTotal != 1000 || p.Currency != "EUR" {
		t.Errorf("unexpected payment: %+v", p)
	}
	if !strings.Contains(p.URL, "https://spectro/pay/1") {
		t.Errorf("redirect URL missing: %s", p.URL)
	}
}

func TestSpectrocoin_UnsupportedCurrency(t *testing.T) {
	t.Parallel()
	provider := New("cb", "ok", "cancel").Spectrocoin("m", "p", genRSAPEM(t))
	if _, err := provider.Pay(Cart{
		Currency: "RUB",
		Items:    []Item{{PriceData: Price{UnitAmount: 1}, Quantity: 1}},
	}); err == nil {
		t.Fatal("expected unsupported currency error")
	}
}

func TestSpectrocoin_InvalidPrivateKey(t *testing.T) {
	t.Parallel()
	provider := New("cb", "ok", "cancel").Spectrocoin("m", "p", "not-a-pem")
	if _, err := provider.Pay(Cart{
		ID: "SC1234567890123", Currency: "EUR",
		Items: []Item{{PriceData: Price{UnitAmount: 100}, Quantity: 1}},
	}); err == nil {
		t.Fatal("expected signing error for invalid key")
	}
}

func TestSpectrocoin_CheckoutIsNoop(t *testing.T) {
	t.Parallel()
	provider := New("cb", "ok", "cancel").Spectrocoin("m", "p", genRSAPEM(t))
	got, err := provider.Checkout(&Payment{}, "sess")
	if err != nil || got != nil {
		t.Errorf("Checkout: got=%v err=%v, want nil,nil", got, err)
	}
}

func TestStatusPayment_Spectrocoin(t *testing.T) {
	t.Parallel()
	cases := map[string]Status{
		"1":   UNPAID,
		"2":   PROCESSED,
		"3":   PAID,
		"4":   FAILED,
		"5":   FAILED,
		"6":   TEST,
		"999": FAILED,
	}
	for in, want := range cases {
		if got := StatusPayment(SPECTROCOIN, in); got != want {
			t.Errorf("Spectrocoin[%s] = %s, want %s", in, got, want)
		}
	}
}
