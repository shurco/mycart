package litepay_test

import (
	"fmt"
	"github.com/shurco/litecart/pkg/litepay"
)

// Example demonstrates basic usage of the litepay package with Stripe.
func Example() {
	// Initialize payment configuration
	pay := litepay.New(
		"https://example.com/payment/callback",
		"https://example.com/payment/success",
		"https://example.com/payment/cancel",
	)

	// Create a shopping cart
	cart := litepay.Cart{
		ID:       "ABC123XYZ456789",
		Currency: "USD",
		Items: []litepay.Item{
			{
				PriceData: litepay.Price{
					UnitAmount: 1999, // $19.99 in cents
					Product: litepay.Product{
						Name:   "Premium Plan",
						Images: []string{"https://example.com/image.jpg"},
					},
				},
				Quantity: 1,
			},
		},
	}

	// Create payment session with Stripe
	stripe := pay.Stripe("sk_test_your_secret_key")
	payment, err := stripe.Pay(cart)
	if err != nil {
		// Expected error without valid API key
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Payment created: %s\n", payment.Status)
	fmt.Printf("Redirect to: %s\n", payment.URL)
	// Output: Error: the server returned an error
}

// ExampleStatusPayment demonstrates status mapping from provider-specific statuses.
func ExampleStatusPayment() {
	// Map Stripe status
	status := litepay.StatusPayment(litepay.STRIPE, "succeeded")
	fmt.Printf("Stripe 'succeeded': %s\n", status)

	// Map PayPal status
	status = litepay.StatusPayment(litepay.PAYPAL, "COMPLETED")
	fmt.Printf("PayPal 'COMPLETED': %s\n", status)

	// Map SpectroCoin status
	status = litepay.StatusPayment(litepay.SPECTROCOIN, "3")
	fmt.Printf("SpectroCoin '3': %s\n", status)

	// Output:
	// Stripe 'succeeded': paid
	// PayPal 'COMPLETED': paid
	// SpectroCoin '3': paid
}

// ExampleCfg_Stripe demonstrates Stripe payment provider usage.
func ExampleCfg_Stripe() {
	pay := litepay.New(
		"https://example.com/callback",
		"https://example.com/success",
		"https://example.com/cancel",
	)

	stripe := pay.Stripe("sk_test_...")

	cart := litepay.Cart{
		ID:       "ORDER1234567890",
		Currency: "USD",
		Items: []litepay.Item{
			{
				PriceData: litepay.Price{
					UnitAmount: 4999, // $49.99
					Product: litepay.Product{
						Name: "E-book: Go Programming",
					},
				},
				Quantity: 1,
			},
		},
	}

	payment, err := stripe.Pay(cart)
	if err != nil {
		// Expected error without valid API key
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Status: %s\n", payment.Status)
	// Output: Error: the server returned an error
}

// ExampleCfg_Paypal demonstrates PayPal payment provider usage.
func ExampleCfg_Paypal() {
	pay := litepay.New(
		"https://example.com/callback",
		"https://example.com/success",
		"https://example.com/cancel",
	)

	paypal := pay.Paypal("client_id", "secret_key")

	cart := litepay.Cart{
		ID:       "ORDER1234567890",
		Currency: "EUR",
		Items: []litepay.Item{
			{
				PriceData: litepay.Price{
					UnitAmount: 2999, // €29.99
					Product: litepay.Product{
						Name: "Monthly Subscription",
					},
				},
				Quantity: 1,
			},
		},
	}

	payment, err := paypal.Pay(cart)
	if err != nil {
		// Expected error without valid API credentials
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Provider: %s\n", payment.PaymentSystem)
	// Output: Error: the server returned an error
}

// ExampleCfg_Dummy demonstrates dummy payment provider usage for testing.
func ExampleCfg_Dummy() {
	pay := litepay.New(
		"https://example.com/callback",
		"https://example.com/success",
		"https://example.com/cancel",
	)

	dummy := pay.Dummy()

	cart := litepay.Cart{
		ID:       "TEST12345678900",
		Currency: "USD",
		Items: []litepay.Item{
			{
				PriceData: litepay.Price{
					UnitAmount: 0, // Free item
					Product: litepay.Product{
						Name: "Free Trial",
					},
				},
				Quantity: 1,
			},
		},
	}

	payment, err := dummy.Pay(cart)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Status: %s\n", payment.Status)
	// Output: Status: paid
}

// ExamplePayment_Validate demonstrates payment validation.
func ExamplePayment_Validate() {
	// Valid payment
	payment := litepay.Payment{
		CartID: "ABC123XYZ456789", // Exactly 15 characters
	}
	err := payment.Validate()
	fmt.Printf("Valid payment: %v\n", err)

	// Invalid payment (too short)
	payment = litepay.Payment{
		CartID: "SHORT",
	}
	err = payment.Validate()
	fmt.Printf("Invalid payment: %v\n", err != nil)

	// Output:
	// Valid payment: <nil>
	// Invalid payment: true
}
