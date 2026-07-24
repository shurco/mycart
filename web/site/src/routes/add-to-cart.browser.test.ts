import { expect, test } from 'vitest'
import { browser } from 'vitest/browser'

// TDD RED Phase: Write failing tests for add to cart functionality
test.describe('Add to Cart from Product List', () => {
  test('should add product to cart from product list page', async () => {
    await browser.url('http://localhost:8080/')

    // Wait for products to load
    const productCard = await browser.$('[data-testid="product-card"]')
    await productCard.waitForExist({ timeout: 10000 })

    // Click add to cart button on first product
    const addButton = await productCard.$('button*=ADD')
    await addButton.click()

    // Button should change to "Remove" state
    await browser.waitUntil(
      async () => {
        const button = await productCard.$('button*=REMOVE')
        return await button.isDisplayed()
      },
      { timeout: 5000 }
    )
  })

  test('should toggle product in cart on repeated clicks', async () => {
    await browser.url('http://localhost:8080/')

    const productCard = await browser.$('[data-testid="product-card"]')
    await productCard.waitForExist({ timeout: 10000 })

    // First click: add to cart
    const addButton = await productCard.$('button*=ADD')
    await addButton.click()

    await browser.waitUntil(
      async () => {
        const removeBtn = await productCard.$('button*=REMOVE')
        return await removeBtn.isDisplayed()
      },
      { timeout: 5000 }
    )

    // Second click: remove from cart
    const removeButton = await productCard.$('button*=REMOVE')
    await removeButton.click()

    await browser.waitUntil(
      async () => {
        const addBtn = await productCard.$('button*=ADD')
        return await addBtn.isDisplayed()
      },
      { timeout: 5000 }
    )
  })
})

test.describe('Add to Cart from Product Detail', () => {
  test('should add product to cart from product detail page', async () => {
    await browser.url('http://localhost:8080/')

    // Navigate to product detail
    const productCard = await browser.$('[data-testid="product-card"]')
    await productCard.waitForExist({ timeout: 10000 })

    const productLink = await productCard.$('a')
    await productLink.click()

    await browser.waitUntil(
      async () => (await browser.getUrl()).includes('/products/'),
      { timeout: 5000 }
    )

    // Add to cart from detail page
    const addButton = await browser.$('button*=ADD')
    await addButton.click()

    // Button should change state
    await browser.waitUntil(
      async () => {
        const removeBtn = await browser.$('button*=REMOVE')
        return await removeBtn.isDisplayed()
      },
      { timeout: 5000 }
    )
  })

  test('should reflect cart state when navigating back to list', async () => {
    await browser.url('http://localhost:8080/')

    // Add product from list
    const productCard = await browser.$('[data-testid="product-card"]')
    await productCard.waitForExist({ timeout: 10000 })

    const addButton = await productCard.$('button*=ADD')
    await addButton.click()

    // Navigate to detail page
    await browser.waitUntil(
      async () => {
        const removeBtn = await productCard.$('button*=REMOVE')
        return await removeBtn.isDisplayed()
      },
      { timeout: 5000 }
    )

    const productLink = await productCard.$('a')
    await productLink.click()

    await browser.waitUntil(
      async () => (await browser.getUrl()).includes('/products/'),
      { timeout: 5000 }
    )

    // Should show "Remove" button
    const removeButton = await browser.$('button*=REMOVE')
    await expect(removeButton.isDisplayed()).resolves.toBe(true)

    // Navigate back
    await browser.back()

    // Should still show "Remove" button in list
    const productCardAgain = await browser.$('[data-testid="product-card"]')
    const removeButtonInList = await productCardAgain.$('button*=REMOVE')
    await expect(removeButtonInList.isDisplayed()).resolves.toBe(true)
  })
})
