package queries

import (
	"context"

	"github.com/shurco/mycart/internal/models"
)

// ValidateCartItems validates cart items against current product data
// Returns validation result with errors if quantity or price mismatches detected
func ValidateCartItems(
	ctx context.Context,
	db *Base,
	requestedProducts []models.CartProduct,
	currency string,
) (*models.CartValidationResult, error) {
	result := &models.CartValidationResult{
		Valid:          true,
		Errors:         nil,
		CorrectedItems: make([]models.CorrectedCartItem, len(requestedProducts)),
	}

	if len(requestedProducts) == 0 {
		return result, nil
	}

	// Extract product IDs for database query
	productIDs := make([]models.CartProduct, len(requestedProducts))
	for i, p := range requestedProducts {
		productIDs[i] = models.CartProduct{ProductID: p.ProductID}
	}

	// Fetch current product data
	// Use private=true to include inactive parent products (variant active status is checked separately)
	products, err := db.ListProducts(ctx, true, 0, 0, "", productIDs...)
	if err != nil {
		return nil, err
	}

	// Build product map for quick lookup
	productMap := make(map[string]*models.Product, len(products.Products))
	for i := range products.Products {
		productMap[products.Products[i].ID] = &products.Products[i]
	}

	// Validate each cart item
	for i, requested := range requestedProducts {
		validationError, correctedItem := validateCartItem(i, requested, productMap)

		result.CorrectedItems[i] = correctedItem

		if validationError != nil {
			result.Valid = false
			result.Errors = append(result.Errors, *validationError)
		}
	}

	return result, nil
}

// validateCartItem validates a single cart item against current product data
func validateCartItem(
	index int,
	requested models.CartProduct,
	productMap map[string]*models.Product,
) (*models.CartValidationError, models.CorrectedCartItem) {
	product, exists := productMap[requested.ProductID]

	// Check if product exists
	if !exists {
		return refuseCartItem(index, requested, requested.VariantID, "product_not_found", 0, 0)
	}

	// A line quantity is the number of copies ordered, and only a positive
	// count is one. A zero or negative value is rejected here rather than
	// carried into the total: multiplied into the per item subtotal it lets a
	// cart cancel a paid item against itself and check out at a total of zero,
	// which the dummy provider then settles without a real payment.
	if requested.Quantity < 1 {
		return refuseCartItem(index, requested, requested.VariantID, "invalid_quantity", 0, 0)
	}

	// Handle variant validation
	// For variant products, the variant's active status is checked, not the parent's
	if requested.VariantID != nil && *requested.VariantID != "" {
		return validateVariantItem(index, requested, product)
	}

	// Check if product requires a variant but none was provided
	if product.HasVariants {
		return refuseCartItem(index, requested, nil, "variant_required", requested.UnitPrice, product.Amount)
	}

	// Check if non-variant product is active
	if !product.Active {
		return refuseCartItem(index, requested, requested.VariantID, "product_inactive", product.Amount, product.Amount)
	}

	// Validate non-variant product
	return validateNonVariantItem(index, requested, product)
}

// chargesStock reports whether a product can run out.
//
// A digital file cannot: the shop stores one file and serves the same bytes to
// every buyer, so `quantity` on such a product is not stock. Reading it as stock
// is what made a shop unable to sell the thing it sells most — a guide left at
// the quantity the panel's form offers by default, which is zero, was answered
// with "quantity unavailable" at every checkout.
//
// Everything else keeps the count it has always been checked against. A licence
// key is stock in the literal sense: each buyer takes one unclaimed row out of
// digital_data, and the operator's number is what stops the shop from promising
// more keys than it holds.
func chargesStock(product *models.Product) bool {
	return product.Digital.Type != models.DigitalFile
}

// orderableQuantity is the most copies of one item an order may hold.
//
// A download is one copy. The shop stores a single file and hands the same
// bytes to every buyer, so a second copy is the same file charged twice: the
// buyer is not getting anything the shop would have to produce. Everything
// else is limited by the stock it keeps.
//
// This is the other half of the same rule. `chargesStock` stops the shop from
// refusing a download it cannot run out of; this stops it from selling more
// copies of one than there are copies to sell.
func orderableQuantity(stock int, product *models.Product) int {
	if !chargesStock(product) {
		return 1
	}
	return stock
}

// validateVariantItem validates a cart item with a variant
func validateVariantItem(
	index int,
	requested models.CartProduct,
	product *models.Product,
) (*models.CartValidationError, models.CorrectedCartItem) {
	var variant *models.ProductVariant

	// Find the variant
	for i := range product.Variants {
		if product.Variants[i].ID == *requested.VariantID {
			variant = &product.Variants[i]
			break
		}
	}

	// Variant not found
	if variant == nil {
		return refuseCartItem(index, requested, requested.VariantID, "product_not_found", product.Amount, product.Amount)
	}

	currentUnitPrice := product.Amount + variant.PriceSurcharge

	// Variant inactive
	if !variant.Active {
		return refuseCartItem(index, requested, requested.VariantID, "product_inactive", currentUnitPrice, currentUnitPrice)
	}

	// Check if unit price changed (when client sends unit_price)
	if requested.UnitPrice > 0 && requested.UnitPrice != currentUnitPrice {
		return reportPriceChange(index, requested, requested.VariantID, variant.Quantity, currentUnitPrice)
	}

	// Check quantity availability
	orderable := orderableQuantity(variant.Quantity, product)
	if requested.Quantity > orderable {
		return reportQuantityUnavailable(index, requested, requested.VariantID, orderable, currentUnitPrice)
	}

	// All validation passed
	return nil, acceptedCartItem(requested, requested.VariantID, currentUnitPrice)
}

// validateNonVariantItem validates a cart item without a variant
func validateNonVariantItem(
	index int,
	requested models.CartProduct,
	product *models.Product,
) (*models.CartValidationError, models.CorrectedCartItem) {
	currentUnitPrice := product.Amount

	// Check if unit price changed (when client sends unit_price)
	if requested.UnitPrice > 0 && requested.UnitPrice != currentUnitPrice {
		return reportPriceChange(index, requested, nil, product.Quantity, currentUnitPrice)
	}

	// Check quantity availability
	orderable := orderableQuantity(product.Quantity, product)
	if requested.Quantity > orderable {
		return reportQuantityUnavailable(index, requested, nil, orderable, currentUnitPrice)
	}

	// All validation passed
	return nil, acceptedCartItem(requested, nil, currentUnitPrice)
}

// refuseCartItem answers a line that cannot be ordered at all: the corrected
// item keeps the current unit price but is unavailable with a quantity of zero.
func refuseCartItem(
	index int,
	requested models.CartProduct,
	variantID *string,
	errorType string,
	requestedUnitPrice, currentUnitPrice int,
) (*models.CartValidationError, models.CorrectedCartItem) {
	return &models.CartValidationError{
		ItemIndex:          index,
		ProductID:          requested.ProductID,
		VariantID:          variantID,
		ErrorType:          errorType,
		RequestedQty:       requested.Quantity,
		AvailableQty:       0,
		RequestedUnitPrice: requestedUnitPrice,
		CurrentUnitPrice:   currentUnitPrice,
		RequestedTotal:     0,
		CurrentTotal:       0,
	}, models.CorrectedCartItem{
		ProductID: requested.ProductID,
		VariantID: variantID,
		Quantity:  0,
		UnitPrice: currentUnitPrice,
		Available: false,
	}
}

// reportPriceChange answers a line whose client-side unit price is stale: the
// corrected item keeps the requested quantity at the current price.
func reportPriceChange(
	index int,
	requested models.CartProduct,
	variantID *string,
	availableQty, currentUnitPrice int,
) (*models.CartValidationError, models.CorrectedCartItem) {
	return &models.CartValidationError{
		ItemIndex:          index,
		ProductID:          requested.ProductID,
		VariantID:          variantID,
		ErrorType:          "price_changed",
		RequestedQty:       requested.Quantity,
		AvailableQty:       availableQty,
		RequestedUnitPrice: requested.UnitPrice,
		CurrentUnitPrice:   currentUnitPrice,
		RequestedTotal:     requested.Quantity * requested.UnitPrice,
		CurrentTotal:       requested.Quantity * currentUnitPrice,
	}, models.CorrectedCartItem{
		ProductID: requested.ProductID,
		VariantID: variantID,
		Quantity:  requested.Quantity,
		UnitPrice: currentUnitPrice,
		Available: true,
	}
}

// reportQuantityUnavailable answers a line that asks for more than can be
// ordered: the corrected item is unavailable with a quantity of zero.
func reportQuantityUnavailable(
	index int,
	requested models.CartProduct,
	variantID *string,
	orderable, currentUnitPrice int,
) (*models.CartValidationError, models.CorrectedCartItem) {
	return &models.CartValidationError{
		ItemIndex:          index,
		ProductID:          requested.ProductID,
		VariantID:          variantID,
		ErrorType:          "quantity_unavailable",
		RequestedQty:       requested.Quantity,
		AvailableQty:       orderable,
		RequestedUnitPrice: currentUnitPrice,
		CurrentUnitPrice:   currentUnitPrice,
		RequestedTotal:     requested.Quantity * currentUnitPrice,
		CurrentTotal:       0,
	}, models.CorrectedCartItem{
		ProductID: requested.ProductID,
		VariantID: variantID,
		Quantity:  0,
		UnitPrice: currentUnitPrice,
		Available: false,
	}
}

// acceptedCartItem is the corrected item of a line that passed validation.
func acceptedCartItem(requested models.CartProduct, variantID *string, currentUnitPrice int) models.CorrectedCartItem {
	return models.CorrectedCartItem{
		ProductID: requested.ProductID,
		VariantID: variantID,
		Quantity:  requested.Quantity,
		UnitPrice: currentUnitPrice,
		Available: true,
	}
}
