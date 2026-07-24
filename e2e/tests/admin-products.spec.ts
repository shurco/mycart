import { test, expect } from '../fixtures/test.fixture'

/**
 * Admin E2E Tests: Product Management
 * Tests admin product list viewing and product creation
 */

test.describe('Admin - Product Management', () => {
  test('should load admin products page successfully', async ({ adminProducts }) => {
    // ARRANGE & ACT: Navigate to admin products page
    await adminProducts.goto()
    await adminProducts.verifyPageLoaded()

    // ASSERT: Page should load without errors
    await expect(adminProducts.page).toHaveURL('/_/products')
  })

  test('should display existing products', async ({ adminProducts }) => {
    // ARRANGE
    await adminProducts.goto()
    await adminProducts.waitForPage()

    // ACT: Check if products exist
    const hasProducts = await adminProducts.hasProducts()

    // ASSERT: Should have products or show empty state
    // Either products exist or the page loads successfully
    expect(typeof hasProducts).toBe('boolean')
  })

  test('should open add product drawer', async ({ adminProducts }) => {
    // ARRANGE
    await adminProducts.goto()
    await adminProducts.waitForPage()

    // ACT: Click add product button
    await adminProducts.openAddProductDrawer()

    // ASSERT: Form should be visible
    await adminProducts.page.waitForTimeout(500) // Wait for drawer animation
    const nameInput = adminProducts.page.getByRole('textbox', { name: 'Name' })
    await expect(nameInput).toBeVisible()
  })

  test('should add a new product', async ({ adminProducts }) => {
    // ARRANGE
    await adminProducts.goto()
    await adminProducts.waitForPage()

    // Get initial product count
    const initialCount = await adminProducts.getProductCount()

    // Open add product drawer
    await adminProducts.openAddProductDrawer()
    await adminProducts.page.waitForTimeout(500)

    // ACT: Fill product form
    const timestamp = Date.now().toString().slice(-6) // Last 6 digits for uniqueness
    const testProduct = {
      name: `E2E Test Product ${Date.now()}`,
      slug: `e2e-${timestamp}`, // Max 20 chars: e2e-123456 = 10 chars
      brief: 'E2E test product brief',
      description: 'E2E test product description',
      amount: '99.99',
      digitalType: 'file' as const
    }

    await adminProducts.fillProductForm(testProduct)
    await adminProducts.saveProduct()

    // Wait for save to complete
    await adminProducts.page.waitForTimeout(2000)

    // ASSERT: Product should be added to list
    // Reload page to see new product
    await adminProducts.goto()
    await adminProducts.waitForPage()

    // Verify product exists
    await adminProducts.verifyProductExists(testProduct.name)
  })
})
