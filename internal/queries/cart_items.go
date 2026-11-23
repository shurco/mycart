package queries

import (
	"github.com/shurco/mycart/internal/models"
)

// BuildCartItems builds cart items with full product information
func BuildCartItems(cart *models.Cart, products *models.Products) []map[string]any {
	if len(cart.Cart) == 0 || len(products.Products) == 0 {
		return nil
	}

	productMap := make(map[string]*models.Product, len(products.Products))
	for i := range products.Products {
		productMap[products.Products[i].ID] = &products.Products[i]
	}

	cartItems := make([]map[string]any, 0, len(cart.Cart))
	for _, cartItem := range cart.Cart {
		product, ok := productMap[cartItem.ProductID]
		if !ok {
			continue
		}

		// Generate unique ID: use product_id + variant_id when variant exists
		itemID := product.ID
		if cartItem.VariantID != nil && *cartItem.VariantID != "" {
			itemID = product.ID + "_" + *cartItem.VariantID
		}

		item := map[string]any{
			"id":       itemID,
			"name":     product.Name,
			"slug":     product.Slug,
			"amount":   product.Amount,
			"quantity": cartItem.Quantity,
		}

		if len(product.Images) > 0 {
			item["image"] = product.Images[0]
		}

		// Include variant information if present
		if cartItem.VariantID != nil && *cartItem.VariantID != "" {
			addVariantDetails(item, product, *cartItem.VariantID)
		}

		cartItems = append(cartItems, item)
	}

	return cartItems
}

// addVariantDetails adds the chosen variant of product to a cart item.
func addVariantDetails(item map[string]any, product *models.Product, variantID string) {
	item["variant_id"] = variantID

	// Find the matching variant details
	for _, variant := range product.Variants {
		if variant.ID == variantID {
			// Include variant option values (e.g. {"Size": "Medium", "Color": "Black"})
			if len(variant.OptionValues) > 0 {
				item["variant_options"] = variant.OptionValues
			}
			// Include SKU if available
			if variant.SKU != "" {
				item["variant_sku"] = variant.SKU
			}
			// Include price surcharge for accurate unit price calculation
			item["variant_price_surcharge"] = variant.PriceSurcharge
			break
		}
	}
}
