import { expect, test } from 'vitest'
import { browser } from 'vitest/browser'

// TDD RED Phase: Write failing tests for delete from cart functionality
test.describe('Delete from Cart', () => {
  test('should remove product from cart page', async () => {
    // First, add a product to cart
    await browser.url('http://localhost:8080/')

    const productCard = await browser.$('[data-testid="product-card"]')
    await productCard.waitForExist({ timeout: 10000 })

    const addButton = await productCard.$('button*=ADD')
    await addButton.click()

    // Navigate to cart
    await browser.url('http://localhost:8080/cart')

    // Find cart items
    const cartItems = await browser.$$('[data-testid="cart-item"]')
    const initialCount = cartItems.length
    expect(initialCount).toBeGreaterThan(0)

    // Click remove/delete button on first item
    const deleteButton = await cartItems[0].$('button*=REMOVE')
    await deleteButton.click()

    // Cart should have one less item
    await browser.pause(500) // Wait for removal
    const remainingItems = await browser.$$('[data-testid="cart-item"]')
    expect(remainingItems.length).toBe(initialCount - 1)
  })

  test('should show empty cart message when all items removed', async () => {
    // Add product to cart
    await browser.url('http://localhost:8080/')

    const productCard = await browser.$('[data-testid="product-card"]')
    await productCard.waitForExist({ timeout: 10000 })

    const addButton = await productCard.$('button*=ADD')
    await addButton.click()

    // Navigate to cart
    await browser.url('http://localhost:8080/cart')

    // Remove all items
    let cartItems = await browser.$$('[data-testid="cart-item"]')

    while (cartItems.length > 0) {
      const deleteButton = await cartItems[0].$('button*=REMOVE')
      await deleteButton.click()
      await browser.pause(500)
      cartItems = await browser.$$('[data-testid="cart-item"]')
    }

    // Should show empty cart message
    const emptyMessage = await browser.$('*=empty')
    await expect(emptyMessage.isDisplayed()).resolves.toBe(true)
  })

  test('should update cart total after removing item', async () => {
    // Add multiple products
    await browser.url('http://localhost:8080/')

    const productCards = await browser.$$('[data-testid="product-card"]')
    const productCount = Math.min(productCards.length, 2)

    for (let i = 0; i < productCount; i++) {
      const addButton = await productCards[i].$('button*=ADD')
      await addButton.click()
      await browser.pause(500)
    }

    // Navigate to cart
    await browser.url('http://localhost:8080/cart')

    // Get initial total
    const totalElement = await browser.$('[data-testid="cart-total"]')
    const initialTotal = await totalElement.getText()

    // Remove one item
    const cartItems = await browser.$$('[data-testid="cart-item"]')
    const deleteButton = await cartItems[0].$('button*=REMOVE')
    await deleteButton.click()

    await browser.pause(500)

    // Total should change
    const newTotal = await totalElement.getText()
    expect(newTotal).not.toBe(initialTotal)
  })
})
