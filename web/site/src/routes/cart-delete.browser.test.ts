import { expect, test } from 'vitest'
import { page } from '@vitest/browser/context'

// TDD RED Phase: Write failing tests for delete from cart functionality
test.describe('Delete from Cart', () => {
  test('should remove product from cart page', async () => {
    // First, add a product to cart
    await page.goto('http://localhost:8080/')
    await page.waitForSelector('[data-testid="product-card"]', { timeout: 10000 })

    const firstProduct = page.getByTestId('product-card').first()
    await firstProduct.getByRole('button', { name: /add/i }).click()

    // Navigate to cart
    await page.goto('http://localhost:8080/cart')

    // Find cart items
    const cartItems = page.getByTestId('cart-item')
    const initialCount = await cartItems.count()
    expect(initialCount).toBeGreaterThan(0)

    // Click remove/delete button on first item
    const firstItem = cartItems.first()
    const deleteButton = firstItem.getByRole('button', { name: /remove|delete/i })
    await deleteButton.click()

    // Cart should have one less item
    const remainingItems = page.getByTestId('cart-item')
    expect(await remainingItems.count()).toBe(initialCount - 1)
  })

  test('should show empty cart message when all items removed', async () => {
    // Add product to cart
    await page.goto('http://localhost:8080/')
    await page.waitForSelector('[data-testid="product-card"]', { timeout: 10000 })

    const firstProduct = page.getByTestId('product-card').first()
    await firstProduct.getByRole('button', { name: /add/i }).click()

    // Navigate to cart
    await page.goto('http://localhost:8080/cart')

    // Remove all items
    const cartItems = page.getByTestId('cart-item')
    const count = await cartItems.count()

    for (let i = 0; i < count; i++) {
      const firstItem = page.getByTestId('cart-item').first()
      const deleteButton = firstItem.getByRole('button', { name: /remove|delete/i })
      await deleteButton.click()
    }

    // Should show empty cart message
    const emptyMessage = page.getByText(/empty|no items/i)
    await expect.element(emptyMessage).toBeVisible()
  })

  test('should update cart total after removing item', async () => {
    // Add multiple products
    await page.goto('http://localhost:8080/')
    await page.waitForSelector('[data-testid="product-card"]', { timeout: 10000 })

    const products = page.getByTestId('product-card')
    const productCount = Math.min(await products.count(), 2)

    for (let i = 0; i < productCount; i++) {
      const product = products.nth(i)
      await product.getByRole('button', { name: /add/i }).click()
    }

    // Navigate to cart
    await page.goto('http://localhost:8080/cart')

    // Get initial total
    const totalElement = page.getByTestId('cart-total')
    const initialTotal = await totalElement.textContent()

    // Remove one item
    const firstItem = page.getByTestId('cart-item').first()
    const deleteButton = firstItem.getByRole('button', { name: /remove|delete/i })
    await deleteButton.click()

    // Total should change
    const newTotal = await totalElement.textContent()
    expect(newTotal).not.toBe(initialTotal)
  })
})
