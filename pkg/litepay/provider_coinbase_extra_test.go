package litepay

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func coinbaseServer(t *testing.T, status string, amount string) *httptest.Server {
	t.Helper()
	body := `{
		"data": {
			"id": "charge_1",
			"hosted_url": "https://coinbase/c/charge_1",
			"pricing": {"local": {"amount": "` + amount + `", "currency": "USD"}},
			"timeline": [{"status": "` + status + `"}]
		}
	}`
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		}
		_, _ = w.Write([]byte(body))
	}))
}

func TestCoinbase_Pay_HappyPath(t *testing.T) {
	srv := coinbaseServer(t, "NEW", "10.00")
	t.Cleanup(srv.Close)
	redirectHTTPClient(t, srv, "https://api.commerce.coinbase.com/")

	provider := New("cb", "ok", "cancel").Coinbase("k")
	p, err := provider.Pay(Cart{
		ID:       "CB12345678901234",
		Currency: "USD",
		Items:    []Item{{PriceData: Price{UnitAmount: 1000}, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if p.MerchantID != "charge_1" {
		t.Errorf("MerchantID = %q", p.MerchantID)
	}
	if p.AmountTotal != 1000 || p.Status != UNPAID || p.URL != "https://coinbase/c/charge_1" {
		t.Errorf("unexpected payment: %+v", p)
	}
}

func TestCoinbase_Pay_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	redirectHTTPClient(t, srv, "https://api.commerce.coinbase.com/")

	provider := New("cb", "ok", "cancel").Coinbase("k")
	if _, err := provider.Pay(Cart{Currency: "USD", Items: []Item{{PriceData: Price{UnitAmount: 1}, Quantity: 1}}}); err == nil {
		t.Fatal("expected server error")
	}
}

func TestCoinbase_Checkout_HappyPath(t *testing.T) {
	srv := coinbaseServer(t, "COMPLETED", "5.00")
	t.Cleanup(srv.Close)
	redirectHTTPClient(t, srv, "https://api.commerce.coinbase.com/")

	provider := New("cb", "ok", "cancel").Coinbase("k")
	p := &Payment{CartID: "CB12345678901234"}
	got, err := provider.Checkout(p, "charge_1")
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	if got.Status != PAID || got.AmountTotal != 500 {
		t.Errorf("unexpected checkout: %+v", got)
	}
}

func TestCoinbase_Checkout_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	redirectHTTPClient(t, srv, "https://api.commerce.coinbase.com/")

	provider := New("cb", "ok", "cancel").Coinbase("k")
	if _, err := provider.Checkout(&Payment{}, "id"); err == nil {
		t.Fatal("expected error")
	}
}
