package litepay

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func paypalFixtureServer(t *testing.T, orderStatus, captureAmount string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/oauth2/token":
			_, _ = w.Write([]byte(`{"access_token":"at-1","token_type":"Bearer"}`))
		case "/v2/checkout/orders":
			_, _ = w.Write([]byte(`{
				"id": "ORDER-1",
				"status": "` + orderStatus + `",
				"links": [
					{"href":"https://p.test/approve","rel":"payer-action","method":"GET"}
				]
			}`))
		case "/v2/checkout/orders/TOKEN-1/capture":
			if captureAmount == "" {
				w.WriteHeader(http.StatusUnprocessableEntity)
				return
			}
			_, _ = w.Write([]byte(`{
				"id": "ORDER-1",
				"status": "COMPLETED",
				"purchase_units": [{
					"payments": {
						"captures": [{
							"amount": {"currency_code":"USD","value":"` + captureAmount + `"}
						}]
					}
				}]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestPaypal_Pay_HappyPath(t *testing.T) {
	srv := paypalFixtureServer(t, "PAYER_ACTION_REQUIRED", "0")
	t.Cleanup(srv.Close)
	redirectHTTPClient(t, srv, "https://api.sandbox.paypal.com/")

	provider := New("cb", "ok", "cancel").Paypal("cid", "sec")
	p, err := provider.Pay(Cart{
		ID:       "PPID12345678901",
		Currency: "USD",
		Items: []Item{
			{PriceData: Price{UnitAmount: 1000}, Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if p.URL != "https://p.test/approve" {
		t.Errorf("url = %s", p.URL)
	}
	if p.AmountTotal != 2000 {
		t.Errorf("amount = %d, want 2000", p.AmountTotal)
	}
	if p.Status != PROCESSED {
		t.Errorf("status = %s", p.Status)
	}
}

func TestPaypal_Checkout_HappyPath(t *testing.T) {
	srv := paypalFixtureServer(t, "PAYER_ACTION_REQUIRED", "25.50")
	t.Cleanup(srv.Close)
	redirectHTTPClient(t, srv, "https://api.sandbox.paypal.com/")

	provider := New("cb", "ok", "cancel").Paypal("cid", "sec")
	p := &Payment{CartID: "PPID12345678901"}
	got, err := provider.Checkout(p, "TOKEN-1")
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	if got.MerchantID != "ORDER-1" || got.AmountTotal != 2550 || got.Status != PAID {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestPaypal_Checkout_422(t *testing.T) {
	srv := paypalFixtureServer(t, "CREATED", "") // captureAmount empty -> 422
	t.Cleanup(srv.Close)
	redirectHTTPClient(t, srv, "https://api.sandbox.paypal.com/")

	provider := New("cb", "ok", "cancel").Paypal("cid", "sec")
	p := &Payment{CartID: "PPID12345678901"}
	if _, err := provider.Checkout(p, "TOKEN-1"); err == nil {
		t.Fatal("expected 422 error")
	}
}

func TestPaypal_AccessToken_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	redirectHTTPClient(t, srv, "https://api.sandbox.paypal.com/")

	provider := New("cb", "ok", "cancel").Paypal("cid", "sec")
	if _, err := provider.Pay(Cart{ID: "PPID12345678901", Currency: "USD", Items: []Item{{PriceData: Price{UnitAmount: 1}, Quantity: 1}}}); err == nil {
		t.Fatal("expected oauth error to bubble")
	}
}

func TestPaypal_UnsupportedCurrency(t *testing.T) {
	t.Parallel()
	provider := New("cb", "ok", "cancel").Paypal("cid", "sec")
	if _, err := provider.Pay(Cart{Currency: "RUB", Items: []Item{{PriceData: Price{UnitAmount: 1}, Quantity: 1}}}); err == nil {
		t.Fatal("expected currency error")
	}
}

func TestStatusPayment_PaypalAndDummy(t *testing.T) {
	t.Parallel()
	if StatusPayment(PAYPAL, "COMPLETED") != PAID {
		t.Error("COMPLETED must map to PAID for PayPal")
	}
	if StatusPayment(PAYPAL, "VOIDED") != CANCELED {
		t.Error("VOIDED must map to CANCELED for PayPal")
	}
	if StatusPayment(DUMMY, "paid") != PAID {
		t.Error("dummy paid mapping")
	}
	if StatusPayment(DUMMY, "other") != FAILED {
		t.Error("unknown dummy status must be FAILED")
	}
}
