package litepay

// PaymentSystem identifies the payment provider.
type PaymentSystem string

const (
	STRIPE      PaymentSystem = "stripe"      // Stripe payment provider (credit/debit cards)
	PAYPAL      PaymentSystem = "paypal"      // PayPal payment provider
	SPECTROCOIN PaymentSystem = "spectrocoin" // SpectroCoin cryptocurrency payment provider
	COINBASE    PaymentSystem = "coinbase"    // Coinbase Commerce cryptocurrency payment provider
	DUMMY       PaymentSystem = "dummy"       // Dummy provider for testing (always succeeds, only for free items)
)
