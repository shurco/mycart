package litepay

import (
	"strings"
	"testing"
)

func TestDummy_PayAndCheckout(t *testing.T) {
	t.Parallel()

	pay := New("https://cb.test", "https://ok.test", "https://cancel.test")
	provider := pay.Dummy()

	cart := Cart{
		ID:       "DUMMY1234567890",
		Currency: "eur",
		Items: []Item{
			{PriceData: Price{UnitAmount: 100}, Quantity: 2},
			{PriceData: Price{UnitAmount: 250}, Quantity: 1},
		},
	}

	p, err := provider.Pay(cart)
	if err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if p.Status != PAID {
		t.Errorf("status = %s, want PAID", p.Status)
	}
	if p.AmountTotal != 100*2+250 {
		t.Errorf("amount = %d, want %d", p.AmountTotal, 450)
	}
	if p.Currency != "EUR" {
		t.Errorf("currency = %s, want EUR", p.Currency)
	}
	if !strings.Contains(p.URL, "https://ok.test") {
		t.Errorf("url = %s, missing successURL", p.URL)
	}

	// Checkout is trivial — verify it stamps the cart id and PAID status.
	p.CartID = cart.ID
	got, err := provider.Checkout(p, "session-unused")
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	if got.Status != PAID || got.MerchantID != "dummy_"+cart.ID {
		t.Errorf("Checkout result unexpected: %+v", got)
	}
}

func TestCart_ValidateLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"exact 15 chars", "123456789012345", false},
		{"too short", "short", true},
		{"too long", "thisistoolongforachartid", true},
		// NOTE: ozzo-validation treats empty as "not provided" and skips the
		// Length rule. That is by design — Payment.Validate only rejects
		// values that are *present but malformed*. Callers must still use
		// validation.Required separately if they want to block empty.
		{"empty (skipped by ozzo)", "", false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := Payment{CartID: tc.id}
			err := p.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate(%q) err=%v wantErr=%v", tc.id, err, tc.wantErr)
			}
		})
	}
}
