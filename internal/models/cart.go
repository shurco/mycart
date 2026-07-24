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
	ProductID string  `json:"id"`
	VariantID *string `json:"variant_id,omitempty"`
	Quantity  int     `json:"quantity"`
	UnitPrice int     `json:"unit_price,omitempty"` // Expected price from client for validation
}

// CartPayment is ...
type CartPayment struct {
	Email    string                `json:"email"`
	Provider litepay.PaymentSystem `json:"provider"`
	Products []CartProduct         `json:"products"`
}

// CartValidationResult contains the result of cart validation
type CartValidationResult struct {
	Valid          bool                  `json:"valid"`
	Errors         []CartValidationError `json:"errors,omitempty"`
	CorrectedItems []CorrectedCartItem   `json:"corrected_items,omitempty"`
}

// CartValidationError describes a specific validation failure for a cart item
type CartValidationError struct {
	ItemIndex          int     `json:"item_index"`
	ProductID          string  `json:"product_id"`
	VariantID          *string `json:"variant_id,omitempty"`
	ErrorType          string  `json:"error_type"`
	RequestedQty       int     `json:"requested_qty"`
	AvailableQty       int     `json:"available_qty"`
	RequestedUnitPrice int     `json:"requested_unit_price"`
	CurrentUnitPrice   int     `json:"current_unit_price"`
	RequestedTotal     int     `json:"requested_total"`
	CurrentTotal       int     `json:"current_total"`
}

// CorrectedCartItem represents the server-corrected version of a cart item
type CorrectedCartItem struct {
	ProductID string  `json:"product_id"`
	VariantID *string `json:"variant_id,omitempty"`
	Quantity  int     `json:"quantity"`
	UnitPrice int     `json:"unit_price"`
	Available bool    `json:"available"`
}
