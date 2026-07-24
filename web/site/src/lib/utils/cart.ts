/**
 * Utility for working with cart
 */

import { cartStore } from '$lib/stores/cart'
import type { Product, CartItem, ProductVariant } from '$lib/types/models'

/**
 * Toggles product in cart (adds or removes)
 * @param product - Product to add/remove
 * @param cartItems - Current cart items to check availability
 * @param selectedVariant - Selected variant (if product has variants)
 * @param quantity - Quantity to add (default: 1)
 */
export function toggleCartItem(
  product: Product,
  cartItems: CartItem[],
  selectedVariant?: ProductVariant | null,
  quantity: number = 1
): void {
  // For products with variants, check if this specific variant is in cart
  const inCart = selectedVariant
    ? cartItems.some((item) => item.id === product.id && item.variant_id === selectedVariant.id)
    : cartItems.some((item) => item.id === product.id && !item.variant_id)

  if (inCart) {
    // Remove the specific variant or product
    if (selectedVariant) {
      cartStore.removeVariant(product.id, selectedVariant.id!)
    } else {
      cartStore.remove(product.id)
    }
  } else {
    const image = product.images?.[0] ? { name: product.images[0].name, ext: product.images[0].ext } : null

    // Calculate final price
    const finalAmount = selectedVariant
      ? product.amount + selectedVariant.price_surcharge
      : product.amount

    // Generate variant display name
    const variantName = selectedVariant
      ? Object.entries(selectedVariant.option_values)
          .map(([key, value]) => `${key}: ${value}`)
          .join(', ')
      : undefined

    const cartItem: CartItem = {
      id: product.id,
      name: product.name,
      slug: product.slug,
      amount: finalAmount,
      quantity: quantity,
      image,
      variant_id: selectedVariant?.id,
      variant_name: variantName
    }

    cartStore.add(cartItem)
  }
}
