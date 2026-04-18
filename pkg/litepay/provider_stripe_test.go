package litepay

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func stripeCart(currency string) Cart {
	return Cart{
		ID:       "STRIPE123456789",
		Currency: currency,
		Items: []Item{
			{PriceData: Price{UnitAmount: 1500, Product: Product{
				Name: "T-Shirt", Images: []string{"https://img/1"},
			}}, Quantity: 2},
		},
	}
}

func TestStripe_Pay_HappyPath(t *testing.T) {
	// NOT t.Parallel — mutates package-level httpClient.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/checkout/sessions" || r.Method != http.MethodPost {
			http.Error(w, "bad path/method", http.StatusBadRequest)
			return
		}
		if user, _, _ := r.BasicAuth(); user != "sk_test_123" {
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{
			"amount_total": 3000,
			"currency": "usd",
			"payment_status": "unpaid",
			"url": "https://checkout.stripe/s/abc"
		}`))
	}))
	t.Cleanup(srv.Close)
	redirectHTTPClient(t, srv, "https://api.stripe.com/")

	pay := New("https://cb.test", "https://ok.test", "https://cancel.test")
	provider := pay.Stripe("sk_test_123")

	p, err := provider.Pay(stripeCart("usd"))
	if err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if p.Status != UNPAID {
		t.Errorf("status = %s, want UNPAID", p.Status)
	}
	if p.AmountTotal != 3000 || p.Currency != "USD" {
		t.Errorf("amount/currency wrong: %+v", p)
	}
	if p.URL != "https://checkout.stripe/s/abc" {
		t.Errorf("url = %s", p.URL)
	}
}

func TestStripe_Pay_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	redirectHTTPClient(t, srv, "https://api.stripe.com/")

	provider := New("cb", "ok", "cancel").Stripe("sk")
	if _, err := provider.Pay(stripeCart("usd")); err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestStripe_Pay_UnsupportedCurrency(t *testing.T) {
	t.Parallel()
	provider := New("cb", "ok", "cancel").Stripe("sk")
	if _, err := provider.Pay(stripeCart("rub")); err == nil {
		t.Fatal("expected unsupported currency error")
	}
}

func TestStripe_Checkout_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/checkout/sessions/sess_42" || r.Method != http.MethodGet {
			http.Error(w, "bad path/method", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{
			"amount_total": 5000,
			"currency": "eur",
			"payment_status": "paid",
			"payment_intent": "pi_42"
		}`))
	}))
	t.Cleanup(srv.Close)
	redirectHTTPClient(t, srv, "https://api.stripe.com/")

	provider := New("cb", "ok", "cancel").Stripe("sk")
	p := &Payment{CartID: "STRIPE123456789"}
	got, err := provider.Checkout(p, "sess_42")
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	if got.Status != PAID || got.MerchantID != "pi_42" || got.AmountTotal != 5000 {
		t.Errorf("unexpected checkout result: %+v", got)
	}
}

func TestStatusPayment_Stripe(t *testing.T) {
	t.Parallel()

	cases := map[string]Status{
		"paid":     PAID,
		"unpaid":   UNPAID,
		"open":     PROCESSED,
		"canceled": CANCELED,
		"unknown":  FAILED,
	}
	for in, want := range cases {
		if got := StatusPayment(STRIPE, in); got != want {
			t.Errorf("StatusPayment(STRIPE, %q) = %s, want %s", in, got, want)
		}
	}
}
