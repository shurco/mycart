import { expect, test } from 'vitest'
import { page } from '@vitest/browser/context'

// TDD RED Phase: Write failing tests for site page views
test.describe('Site Page Views', () => {
  test('should display home page with products', async () => {
    await page.goto('http://localhost:8080/')

    // Wait for products to load
    await page.waitForSelector('[data-testid="product-card"]', { timeout: 10000 })

    // Check if page title/heading exists
    const heading = page.getByRole('heading', { name: /products/i })
    await expect.element(heading).toBeVisible()

    // Check if at least one product card is displayed
    const productCards = page.getByTestId('product-card')
    expect(await productCards.count()).toBeGreaterThan(0)
  })

  test('should navigate to product detail page', async () => {
    await page.goto('http://localhost:8080/')

    // Wait for products to load
    await page.waitForSelector('[data-testid="product-card"]', { timeout: 10000 })

    // Click on first product
    const firstProduct = page.getByTestId('product-card').first()
    await firstProduct.click()

    // Should be on product detail page
    await page.waitForURL(/\/products\/.*/, { timeout: 5000 })

    // Check if product detail page elements are present
    const addToCartButton = page.getByRole('button', { name: /add to cart/i })
    await expect.element(addToCartButton).toBeVisible()
  })

  test('should display cart page', async () => {
    await page.goto('http://localhost:8080/cart')

    // Check if cart page loads
    const cartHeading = page.getByRole('heading', { name: /cart|shopping cart/i })
    await expect.element(cartHeading).toBeVisible()
  })

  test('should navigate to cart from header', async () => {
    await page.goto('http://localhost:8080/')

    // Find and click cart link in navigation
    const cartLink = page.getByRole('link', { name: /cart/i })
    await cartLink.click()

    // Should be on cart page
    await page.waitForURL('/cart', { timeout: 5000 })
    expect(page.url()).toContain('/cart')
  })
})
