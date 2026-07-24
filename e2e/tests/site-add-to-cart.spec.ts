import { test, expect } from '../fixtures/test.fixture'

/**
 * Site E2E Tests: Add to Cart Functionality
 * Tests adding products from both list and detail pages
 */

test.describe('Site - Add to Cart', () => {
  test('should add product to cart from product list page', async ({ productList, cart }) => {
    // ARRANGE: Navigate to product list
    await productList.goto()
    await productList.waitForProducts()

    // Get product name before adding to cart
    const productName = await productList.getProductName(0)

    // ACT: Add first product to cart
    await productList.addToCartByIndex(0)

    // ASSERT: Product should show as in cart
    await productList.verifyInCart(0)

    // Verify in cart page
    await cart.goto()
    await cart.verifyCartHasItems(1)
    const cartItemName = await cart.getItemNameByIndex(0)
    expect(cartItemName).toBe(productName)
  })

  test('should add product to cart from product detail page', async ({ page, productList, productDetail, cart }) => {
    // ARRANGE: Get a product slug from list page
    await productList.goto()
    await productList.waitForProducts()

    // Click on first product to go to detail page
    const firstProduct = await productList.getProductByIndex(0)
    await firstProduct.locator('a').first().click()
    await page.waitForURL(/\/products\/.*/)

    // Get current URL to extract slug
    const url = page.url()
    const slug = url.split('/products/')[1]

    // Get product name from detail page
    const productName = await productDetail.getProductName()

    // ACT: Add to cart from detail page
    await productDetail.addToCart()

    // ASSERT: Verify shows as in cart on detail page
    await productDetail.verifyInCart()

    // Verify in cart page
    await cart.goto()
    await cart.verifyCartHasItems(1)
    const cartItemName = await cart.getItemNameByIndex(0)
    expect(cartItemName).toBe(productName)
  })

  test('should add multiple products to cart', async ({ productList, cart }) => {
    // ARRANGE
    await productList.goto()
    await productList.waitForProducts()

    // ACT: Add first two products
    await productList.addToCartByIndex(0)
    await productList.addToCartByIndex(1)

    // ASSERT: Both should be in cart
    await productList.verifyInCart(0)
    await productList.verifyInCart(1)

    // Verify cart has 2 items
    await cart.goto()
    await cart.verifyCartHasItems(2)
  })
})
