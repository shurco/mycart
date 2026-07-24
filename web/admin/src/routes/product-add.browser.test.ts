import { expect, test } from 'vitest'
import { page } from '@vitest/browser/context'

// TDD RED Phase: Write failing tests for admin product add functionality
test.describe('Admin Product Add', () => {
  test('should open product add drawer when clicking add button', async () => {
    await page.goto('http://localhost:8080/admin/products')

    // Skip if not authenticated
    if (page.url().includes('/signin')) {
      test.skip()
    }

    // Find and click the "Add Product" button
    const addButton = page.getByRole('button', { name: /add product|new product|\+/i })
    await addButton.click()

    // Drawer should open with form fields
    const nameInput = page.getByLabel(/name/i)
    await expect.element(nameInput).toBeVisible()

    const slugInput = page.getByLabel(/slug/i)
    await expect.element(slugInput).toBeVisible()
  })

  test('should validate required fields when adding product', async () => {
    await page.goto('http://localhost:8080/admin/products')

    if (page.url().includes('/signin')) {
      test.skip()
    }

    // Open add product drawer
    const addButton = page.getByRole('button', { name: /add product|new product|\+/i })
    await addButton.click()

    // Try to save without filling required fields
    const saveButton = page.getByRole('button', { name: /save|create|add/i })
    await saveButton.click()

    // Should show validation errors
    const errorMessage = page.getByText(/required|invalid/i)
    await expect.element(errorMessage).toBeVisible()
  })

  test('should successfully add a new product', async () => {
    await page.goto('http://localhost:8080/admin/products')

    if (page.url().includes('/signin')) {
      test.skip()
    }

    // Get initial product count
    const productRows = page.getByTestId('product-row')
    const initialCount = await productRows.count()

    // Open add product drawer
    const addButton = page.getByRole('button', { name: /add product|new product|\+/i })
    await addButton.click()

    // Fill in product details
    const timestamp = Date.now()
    const nameInput = page.getByLabel(/name/i)
    await nameInput.fill(`Test Product ${timestamp}`)

    const slugInput = page.getByLabel(/slug/i)
    await slugInput.fill(`test-product-${timestamp}`)

    const briefInput = page.getByLabel(/brief|description/i).first()
    await briefInput.fill('Test brief description')

    const amountInput = page.getByLabel(/price|amount/i)
    await amountInput.fill('10.00')

    // Save the product
    const saveButton = page.getByRole('button', { name: /save|create|add/i })
    await saveButton.click()

    // Wait for drawer to close and product list to update
    await page.waitForTimeout(1000)

    // Product count should increase
    const updatedProductRows = page.getByTestId('product-row')
    const newCount = await updatedProductRows.count()
    expect(newCount).toBeGreaterThan(initialCount)

    // New product should appear in the list
    const productName = page.getByText(`Test Product ${timestamp}`)
    await expect.element(productName).toBeVisible()
  })

  test('should close drawer without saving on cancel', async () => {
    await page.goto('http://localhost:8080/admin/products')

    if (page.url().includes('/signin')) {
      test.skip()
    }

    // Get initial product count
    const initialProductRows = page.getByTestId('product-row')
    const initialCount = await initialProductRows.count()

    // Open add product drawer
    const addButton = page.getByRole('button', { name: /add product|new product|\+/i })
    await addButton.click()

    // Fill in some data
    const nameInput = page.getByLabel(/name/i)
    await nameInput.fill('Test Product Cancelled')

    // Click cancel button
    const cancelButton = page.getByRole('button', { name: /cancel|close/i })
    await cancelButton.click()

    // Product count should remain the same
    await page.waitForTimeout(500)
    const productRows = page.getByTestId('product-row')
    expect(await productRows.count()).toBe(initialCount)
  })
})
