import { test, expect } from '../fixtures/test.fixture'

/**
 * Site E2E Tests: Cart Item Deletion
 * Tests removing products from cart page
 */

test.describe('Site - Delete from Cart', () => {
  test('should remove product from cart', async ({ productList, cart }) => {
    // ARRANGE: Add product to cart first
    await productList.goto()
    await productList.waitForProducts()
    const productName = await productList.getProductName(0)
    await productList.addToCartByIndex(0)

    // Verify product was added to cart
    await productList.verifyInCart(0)

    // Navigate to cart
    await cart.goto()
    await cart.verifyCartHasItems(1)

    // ACT: Remove the product
    await cart.removeItemByIndex(0)

    // ASSERT: Cart should be empty
    // Wait a bit for the item to be removed
    await cart.page.waitForTimeout(500)
    await cart.verifyCartIsEmpty()
  })

  test('should remove specific product from cart with multiple items', async ({ productList, cart }) => {
    // ARRANGE: Add two products to cart
    await productList.goto()
    await productList.waitForProducts()

    const firstProductName = await productList.getProductName(0)
    const secondProductName = await productList.getProductName(1)

    await productList.addToCartByIndex(0)
    await productList.addToCartByIndex(1)

    // Verify both products were added to cart
    await productList.verifyInCart(0)
    await productList.verifyInCart(1)

    // Navigate to cart
    await cart.goto()
    await cart.verifyCartHasItems(2)

    // ACT: Remove first product
    await cart.removeItemByIndex(0)

    // ASSERT: Only second product should remain
    await cart.page.waitForTimeout(500)
    await cart.verifyCartHasItems(1)
    const remainingItemName = await cart.getItemNameByIndex(0)
    expect(remainingItemName).toBe(secondProductName)
  })

  test('should clear entire cart by removing all items', async ({ productList, cart }) => {
    // ARRANGE: Add two products
    await productList.goto()
    await productList.waitForProducts()
    await productList.addToCartByIndex(0)
    await productList.addToCartByIndex(1)

    // Verify both products were added to cart
    await productList.verifyInCart(0)
    await productList.verifyInCart(1)

    await cart.goto()
    await cart.verifyCartHasItems(2)

    // ACT: Remove both items
    await cart.removeItemByIndex(0)
    await cart.page.waitForTimeout(500)
    await cart.removeItemByIndex(0) // After first removal, second item becomes index 0

    // ASSERT: Cart should be empty
    await cart.page.waitForTimeout(500)
    await cart.verifyCartIsEmpty()
  })
})
