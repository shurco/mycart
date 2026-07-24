import { expect, test } from 'vitest'
import { browser } from 'vitest/browser'

// TDD RED Phase: Write failing tests for admin product add functionality
test.describe('Admin Product Add', () => {
  test('should open product add drawer when clicking add button', async () => {
    await browser.url('http://localhost:8080/admin/products')

    const currentUrl = await browser.getUrl()
    if (currentUrl.includes('/signin')) {
      test.skip()
    }

    // Find and click the "Add Product" button
    const addButton = await browser.$('button*=Add Product')
    await addButton.click()

    // Drawer should open with form fields
    await browser.pause(500)
    const nameInput = await browser.$('input[name="name"]')
    await expect(nameInput.isDisplayed()).resolves.toBe(true)
  })

  test('should validate required fields when adding product', async () => {
    await browser.url('http://localhost:8080/admin/products')

    const currentUrl = await browser.getUrl()
    if (currentUrl.includes('/signin')) {
      test.skip()
    }

    // Open add product drawer
    const addButton = await browser.$('button*=Add Product')
    await addButton.click()

    await browser.pause(500)

    // Try to save without filling required fields
    const saveButton = await browser.$('button*=Save')
    await saveButton.click()

    await browser.pause(500)

    // Should show validation errors (toast or inline)
    const hasError = await browser.execute(() => {
      // Check for toast messages or error text
      return document.body.textContent?.includes('required') ||
             document.body.textContent?.includes('invalid')
    })
    expect(hasError).toBe(true)
  })

  test('should successfully add a new product', async () => {
    await browser.url('http://localhost:8080/admin/products')

    const currentUrl = await browser.getUrl()
    if (currentUrl.includes('/signin')) {
      test.skip()
    }

    // Get initial product count
    const productRows = await browser.$$('[data-testid="product-row"]')
    const initialCount = productRows.length

    // Open add product drawer
    const addButton = await browser.$('button*=Add Product')
    await addButton.click()

    await browser.pause(500)

    // Fill in product details
    const timestamp = Date.now()
    const nameInput = await browser.$('input[name="name"]')
    await nameInput.setValue(`Test Product ${timestamp}`)

    const slugInput = await browser.$('input[name="slug"]')
    await slugInput.setValue(`test-product-${timestamp}`)

    const briefInput = await browser.$('textarea[name="brief"]')
    await briefInput.setValue('Test brief description')

    const amountInput = await browser.$('input[name="amount"]')
    await amountInput.setValue('10.00')

    // Save the product
    const saveButton = await browser.$('button*=Save')
    await saveButton.click()

    // Wait for drawer to close and product list to update
    await browser.pause(2000)

    // Product count should increase
    const updatedProductRows = await browser.$$('[data-testid="product-row"]')
    expect(updatedProductRows.length).toBeGreaterThan(initialCount)
  })

  test('should close drawer without saving on cancel', async () => {
    await browser.url('http://localhost:8080/admin/products')

    const currentUrl = await browser.getUrl()
    if (currentUrl.includes('/signin')) {
      test.skip()
    }

    // Get initial product count
    const initialProductRows = await browser.$$('[data-testid="product-row"]')
    const initialCount = initialProductRows.length

    // Open add product drawer
    const addButton = await browser.$('button*=Add Product')
    await addButton.click()

    await browser.pause(500)

    // Fill in some data
    const nameInput = await browser.$('input[name="name"]')
    await nameInput.setValue('Test Product Cancelled')

    // Click cancel or close button
    const closeButton = await browser.$('button[aria-label="Close"]')
    if (await closeButton.isExisting()) {
      await closeButton.click()
    }

    await browser.pause(500)

    // Product count should remain the same
    const productRows = await browser.$$('[data-testid="product-row"]')
    expect(productRows.length).toBe(initialCount)
  })
})
