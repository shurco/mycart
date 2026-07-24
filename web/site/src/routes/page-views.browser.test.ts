import { expect, test } from 'vitest'
import { browser } from 'vitest/browser'

// TDD RED Phase: Write failing tests for site page views
test.describe('Site Page Views', () => {
  test('should display home page with products', async () => {
    await browser.url('http://localhost:8080/')

    // Wait for products to load
    const productCard = await browser.$('[data-testid="product-card"]')
    await productCard.waitForExist({ timeout: 10000 })

    // Check if page title/heading exists
    const heading = await browser.$('h2*=Products')
    await expect(heading.isDisplayed()).resolves.toBe(true)

    // Check if at least one product card is displayed
    const productCards = await browser.$$('[data-testid="product-card"]')
    expect(productCards.length).toBeGreaterThan(0)
  })

  test('should navigate to product detail page', async () => {
    await browser.url('http://localhost:8080/')

    // Wait for products to load
    const productCard = await browser.$('[data-testid="product-card"]')
    await productCard.waitForExist({ timeout: 10000 })

    // Click on first product link
    const firstProductLink = await productCard.$('a')
    await firstProductLink.click()

    // Should be on product detail page
    await browser.waitUntil(
      async () => (await browser.getUrl()).includes('/products/'),
      { timeout: 5000 }
    )

    // Check if product detail page elements are present
    const addToCartButton = await browser.$('button*=Add')
    await expect(addToCartButton.isDisplayed()).resolves.toBe(true)
  })

  test('should display cart page', async () => {
    await browser.url('http://localhost:8080/cart')

    // Check if cart page loads
    const cartHeading = await browser.$('h1')
    await expect(cartHeading.isDisplayed()).resolves.toBe(true)
  })

  test('should navigate to cart from header', async () => {
    await browser.url('http://localhost:8080/')

    // Find and click cart link in navigation
    const cartLink = await browser.$('a[href="/cart"]')
    await cartLink.click()

    // Should be on cart page
    await browser.waitUntil(
      async () => (await browser.getUrl()).includes('/cart'),
      { timeout: 5000 }
    )
    expect(await browser.getUrl()).toContain('/cart')
  })
})
