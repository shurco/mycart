package queries

import (
	"encoding/json"
	"testing"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/litepay"
)

func TestCart_AddUpdateListAndFetch(t *testing.T) {
	db, ctx := bootstrap(t)

	cart := &models.Cart{
		Core:          models.Core{ID: "cart-1"},
		Email:         "buyer@example.com",
		AmountTotal:   123,
		Currency:      "USD",
		PaymentStatus: litepay.UNPAID,
		PaymentSystem: "dummy",
		Cart: []models.CartProduct{
			{ProductID: "p1", Quantity: 1},
		},
	}
	if err := db.AddCart(ctx, cart); err != nil {
		t.Fatalf("AddCart: %v", err)
	}

	got, err := db.Cart(ctx, "cart-1")
	if err != nil {
		t.Fatalf("Cart: %v", err)
	}
	if got.Email != "buyer@example.com" || got.AmountTotal != 123 {
		t.Errorf("unexpected cart: %+v", got)
	}
	if len(got.Cart) != 1 || got.Cart[0].ProductID != "p1" {
		t.Errorf("cart products not deserialised: %+v", got.Cart)
	}

	carts, total, err := db.Carts(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Carts: %v", err)
	}
	if total == 0 || len(carts) == 0 {
		t.Error("expected at least one cart in list")
	}

	cart.PaymentID = "pay-42"
	cart.PaymentStatus = litepay.PAID
	if err := db.UpdateCart(ctx, cart); err != nil {
		t.Fatalf("UpdateCart: %v", err)
	}

	got, _ = db.Cart(ctx, "cart-1")
	if got.PaymentID != "pay-42" || got.PaymentStatus != litepay.PAID {
		t.Errorf("UpdateCart did not persist: %+v", got)
	}
}

func TestCart_NotFound(t *testing.T) {
	db, ctx := bootstrap(t)
	if _, err := db.Cart(ctx, "unknown-cart"); err == nil {
		t.Error("expected error for unknown cart")
	}
}

func TestCartLetterPayment(t *testing.T) {
	db, ctx := bootstrap(t)

	// The letter template must be a valid Letter JSON or the Unmarshal fails.
	tpl, _ := json.Marshal(models.Letter{
		Subject: "Pay please",
		Text:    "Follow {Payment_URL}",
	})
	if err := db.UpdateSettingByKey(ctx, &models.SettingName{
		Key:   "mail_letter_payment",
		Value: string(tpl),
	}); err != nil {
		t.Fatalf("seed template: %v", err)
	}
	if err := db.UpdateSettingByKey(ctx, &models.SettingName{
		Key:   "site_name",
		Value: "Litecart",
	}); err != nil {
		t.Fatalf("seed site name: %v", err)
	}

	mail, err := db.CartLetterPayment(ctx, "buyer@example.com", "1.00 USD", "https://pay.example/abc")
	if err != nil {
		t.Fatalf("CartLetterPayment: %v", err)
	}
	if mail.To != "buyer@example.com" {
		t.Errorf("unexpected To: %s", mail.To)
	}
	if mail.Data["Payment_URL"] != "https://pay.example/abc" {
		t.Errorf("Payment_URL missing: %+v", mail.Data)
	}
	if mail.Data["Site_Name"] != "Litecart" {
		t.Errorf("Site_Name missing: %+v", mail.Data)
	}
}

func TestCartLetterPurchase_CartNotFound(t *testing.T) {
	db, ctx := bootstrap(t)
	if _, err := db.CartLetterPurchase(ctx, "missing-cart"); err == nil {
		t.Error("expected error for missing cart")
	}
}

func TestCartLetterPurchase_HappyPath_DataType(t *testing.T) {
	db, ctx := bootstrap(t)

	p, err := db.AddProduct(ctx, validProductInput())
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	if _, err := db.AddDigitalData(ctx, p.ID, "SECRET-1"); err != nil {
		t.Fatalf("AddDigitalData: %v", err)
	}

	cart := &models.Cart{
		Core:          models.Core{ID: "cart-p1"},
		Email:         "buyer@example.com",
		Currency:      "USD",
		AmountTotal:   1000,
		PaymentStatus: litepay.PAID,
		PaymentSystem: "dummy",
		Cart:          []models.CartProduct{{ProductID: p.ID, Quantity: 1}},
	}
	if err := db.AddCart(ctx, cart); err != nil {
		t.Fatalf("AddCart: %v", err)
	}

	tpl, _ := json.Marshal(models.Letter{Subject: "Your order", Text: "{Purchases}"})
	if err := db.UpdateSettingByKey(ctx, &models.SettingName{
		Key:   "mail_letter_purchase",
		Value: string(tpl),
	}); err != nil {
		t.Fatalf("seed template: %v", err)
	}
	if err := db.UpdateSettingByKey(ctx, &models.SettingName{
		Key:   "email",
		Value: "admin@example.com",
	}); err != nil {
		t.Fatalf("seed email: %v", err)
	}

	mail, err := db.CartLetterPurchase(ctx, "cart-p1")
	if err != nil {
		t.Fatalf("CartLetterPurchase: %v", err)
	}
	if mail.To != "buyer@example.com" {
		t.Errorf("unexpected To: %s", mail.To)
	}
	if mail.Data["Admin_Email"] != "admin@example.com" {
		t.Errorf("Admin_Email missing: %+v", mail.Data)
	}
	if mail.Data["Purchases"] == "" {
		t.Errorf("Purchases empty: %+v", mail.Data)
	}
}

func TestValidateCartItems_Success(t *testing.T) {
	db, ctx := bootstrap(t)

	// Create test product
	product := &models.Product{
		Core:     models.Core{ID: "test-prod-1"},
		Name:     "Test Product",
		Amount:   2999,
		Quantity: 10,
		Active:   true,
		Slug:     "test-product",
		Digital:  models.Digital{Type: "file"},
	}
	if _, err := db.AddProduct(ctx, product); err != nil {
		t.Fatalf("AddProduct failed: %v", err)
	}

	// Activate product and set quantity
	if err := db.UpdateActive(ctx, product.ID); err != nil {
		t.Fatalf("UpdateActive failed: %v", err)
	}
	if _, err := db.ProductQueries.DB.ExecContext(ctx, "UPDATE product SET quantity = ? WHERE id = ?", 10, product.ID); err != nil {
		t.Fatalf("Update quantity failed: %v", err)
	}

	// Verify product is queryable
	testList, err := db.ListProducts(ctx, false, 0, 0, "", models.CartProduct{ProductID: product.ID, Quantity: 1})
	if err != nil {
		t.Fatalf("ListProducts verification failed: %v", err)
	}
	if len(testList.Products) == 0 {
		t.Fatalf("Product not found in public listing after activation")
	}

	// Validate with available quantity
	cartProducts := []models.CartProduct{
		{ProductID: product.ID, Quantity: 5},
	}

	result, err := ValidateCartItems(ctx, db, cartProducts, "USD")
	if err != nil {
		t.Fatalf("ValidateCartItems error: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid result, got invalid with errors: %v", result.Errors)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got: %d", len(result.Errors))
	}
}

func TestValidateCartItems_QuantityUnavailable(t *testing.T) {
	db, ctx := bootstrap(t)

	// Create product with limited quantity
	product := &models.Product{
		Core:     models.Core{ID: "test-prod-2"},
		Name:     "Limited Product",
		Amount:   1999,
		Quantity: 3,
		Active:   true,
		Slug:     "limited-product",
		Digital:  models.Digital{Type: "file"},
	}
	if _, err := db.AddProduct(ctx, product); err != nil {
		t.Fatalf("AddProduct failed: %v", err)
	}

	// Activate product and set quantity
	if err := db.UpdateActive(ctx, product.ID); err != nil {
		t.Fatalf("UpdateActive failed: %v", err)
	}
	if _, err := db.ProductQueries.DB.ExecContext(ctx, "UPDATE product SET quantity = ? WHERE id = ?", 3, product.ID); err != nil {
		t.Fatalf("Update quantity failed: %v", err)
	}

	// Verify product is queryable
	testList, err := db.ListProducts(ctx, false, 0, 0, "", models.CartProduct{ProductID: product.ID, Quantity: 1})
	if err != nil {
		t.Fatalf("ListProducts verification failed: %v", err)
	}
	if len(testList.Products) == 0 {
		t.Fatalf("Product not found in public listing after activation")
	}

	// Request more than available
	cartProducts := []models.CartProduct{
		{ProductID: product.ID, Quantity: 10},
	}

	result, err := ValidateCartItems(ctx, db, cartProducts, "USD")
	if err != nil {
		t.Fatalf("ValidateCartItems error: %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid result, got valid")
	}

	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got: %d", len(result.Errors))
	}

	if result.Errors[0].ErrorType != "quantity_unavailable" {
		t.Errorf("Expected error type 'quantity_unavailable', got: %s (details: %+v)", result.Errors[0].ErrorType, result.Errors[0])
	}

	if result.CorrectedItems[0].Available {
		t.Error("Expected item to be marked as unavailable")
	}
}

func TestValidateCartItems_ProductNotFound(t *testing.T) {
	db, ctx := bootstrap(t)

	// Request non-existent product
	cartProducts := []models.CartProduct{
		{ProductID: "nonexistent", Quantity: 1},
	}

	result, err := ValidateCartItems(ctx, db, cartProducts, "USD")
	if err != nil {
		t.Fatalf("ValidateCartItems error: %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid result, got valid")
	}

	if len(result.Errors) != 1 {
		t.Fatalf("Expected 1 error, got: %d", len(result.Errors))
	}

	if result.Errors[0].ErrorType != "product_not_found" {
		t.Errorf("Expected error type 'product_not_found', got: %s", result.Errors[0].ErrorType)
	}
}
