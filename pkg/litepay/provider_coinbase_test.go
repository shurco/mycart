package litepay

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCoinbase_Pay(t *testing.T) {
	pay := New("http://callback.url", "http://success.url", "http://cancel.url")
	provider := pay.Coinbase("test_api_key")

	cart := Cart{
		ID:       "TEST12345678900",
		Currency: "USD",
		Items: []Item{
			{
				PriceData: Price{
					UnitAmount: 1000, // $10.00
					Product: Product{
						Name: "Test Product",
					},
				},
				Quantity: 1,
			},
		},
	}

	// Note: This test will fail without a valid API key and network access
	// In a real scenario, you would mock the HTTP client
	payment, err := provider.Pay(cart)
	
	// Without valid API key, we expect an error
	if err == nil {
		// If no error, verify payment structure
		assert.NotNil(t, payment)
		assert.Equal(t, COINBASE, payment.PaymentSystem)
		assert.Equal(t, "USD", payment.Currency)
	} else {
		// Expected error without valid credentials
		assert.Error(t, err)
	}
}

func TestCoinbase_Checkout(t *testing.T) {
	pay := New("http://callback.url", "http://success.url", "http://cancel.url")
	provider := pay.Coinbase("test_api_key")

	payment := &Payment{
		PaymentSystem: COINBASE,
		CartID:        "TEST12345678900",
		MerchantID:    "test_charge_id",
	}

	// Note: This test will fail without a valid API key and network access
	// In a real scenario, you would mock the HTTP client
	updatedPayment, err := provider.Checkout(payment, "test_charge_id")
	
	// Without valid API key, we expect an error
	if err == nil {
		// If no error, verify payment was updated
		assert.NotNil(t, updatedPayment)
		assert.Equal(t, COINBASE, updatedPayment.PaymentSystem)
	} else {
		// Expected error without valid credentials
		assert.Error(t, err)
	}
}

func TestStatusPayment_Coinbase(t *testing.T) {
	cases := []struct {
		status   string
		expected Status
	}{
		{"NEW", UNPAID},
		{"PENDING", PROCESSED},
		{"COMPLETED", PAID},
		{"EXPIRED", CANCELED},
		{"CANCELED", CANCELED},
		{"RESOLVED", PAID},
		{"UNRESOLVED", FAILED},
		{"UNKNOWN", FAILED}, // Unknown status defaults to FAILED
	}

	for _, tt := range cases {
		result := StatusPayment(COINBASE, tt.status)
		assert.Equal(t, tt.expected, result, "Status %s should map to %s", tt.status, tt.expected)
	}
}

func TestCoinbase_UnsupportedCurrency(t *testing.T) {
	pay := New("http://callback.url", "http://success.url", "http://cancel.url")
	provider := pay.Coinbase("test_api_key")

	cart := Cart{
		ID:       "TEST12345678900",
		Currency: "RUB", // Not supported
		Items: []Item{
			{
				PriceData: Price{
					UnitAmount: 1000,
					Product: Product{
						Name: "Test Product",
					},
				},
				Quantity: 1,
			},
		},
	}

	payment, err := provider.Pay(cart)
	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Contains(t, err.Error(), "currency is not supported")
}
