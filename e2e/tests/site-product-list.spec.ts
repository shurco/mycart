import { test, expect } from '../fixtures/test.fixture'

/**
 * Site E2E Tests: Product List Page
 * TDD Approach: RED - Write failing test, GREEN - Verify implementation passes
 */

test.describe('Site - Product List Page', () => {
  test('should load product list page successfully', async ({ productList }) => {
    // ARRANGE: Navigate to product list
    await productList.goto()

    // ACT: Wait for page to load
    await productList.verifyPageLoaded()

    // ASSERT: Products should be visible
    const count = await productList.getProductCount()
    expect(count).toBeGreaterThan(0)
  })

  test('should display product information', async ({ productList }) => {
    // ARRANGE
    await productList.goto()
    await productList.waitForProducts()

    // ACT: Get first product name
    const productName = await productList.getProductName(0)

    // ASSERT: Product name should not be empty
    expect(productName).toBeTruthy()
    expect(productName.length).toBeGreaterThan(0)
  })
})
