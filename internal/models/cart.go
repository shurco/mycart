package models

import "github.com/shurco/mycart/pkg/litepay"

// Cart is ...
type Cart struct {
	Core
	Email         string                `json:"email"`
	Cart          []CartProduct         `json:"cart,omitempty"`
	AmountTotal   int                   `json:"amount_total"`
	Currency      string                `json:"currency"`
	PaymentID     string                `json:"payment_id"`
	PaymentStatus litepay.Status        `json:"payment_status"`
	PaymentSystem litepay.PaymentSystem `json:"payment_system"`
}

// CartProduct is ...
type CartProduct struct {
	ProductID string `json:"id"`
	Quantity  int    `json:"quantity"`
}

// CartPayment is ...
type CartPayment struct {
	Email    string                `json:"email"`
	Provider litepay.PaymentSystem `json:"provider"`
	Products []CartProduct         `json:"products"`
}
