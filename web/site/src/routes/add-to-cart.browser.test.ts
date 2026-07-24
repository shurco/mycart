import { expect, test } from 'vitest'
import { page } from '@vitest/browser/context'

// TDD RED Phase: Write failing tests for add to cart functionality
test.describe('Add to Cart from Product List', () => {
  test('should add product to cart from product list page', async () => {
    await page.goto('http://localhost:8080/')

    // Wait for products to load
    await page.waitForSelector('[data-testid="product-card"]', { timeout: 10000 })

    // Get initial cart count (if displayed)
    const cartBadge = page.locator('[data-testid="cart-count"]')
    const initialCount = (await cartBadge.isVisible())
      ? parseInt(await cartBadge.textContent() || '0')
      : 0

    // Click add to cart button on first product
    const firstProduct = page.getByTestId('product-card').first()
    const addButton = firstProduct.getByRole('button', { name: /add/i })
    await addButton.click()

    // Cart count should increase
    if (await cartBadge.isVisible()) {
      const newCount = parseInt(await cartBadge.textContent() || '0')
      expect(newCount).toBe(initialCount + 1)
    }

    // Button should change to "Remove" state
    const removeButton = firstProduct.getByRole('button', { name: /remove/i })
    await expect.element(removeButton).toBeVisible()
  })

  test('should toggle product in cart on repeated clicks', async () => {
    await page.goto('http://localhost:8080/')

    await page.waitForSelector('[data-testid="product-card"]', { timeout: 10000 })

    const firstProduct = page.getByTestId('product-card').first()

    // First click: add to cart
    await firstProduct.getByRole('button', { name: /add/i }).click()
    await expect.element(firstProduct.getByRole('button', { name: /remove/i })).toBeVisible()

    // Second click: remove from cart
    await firstProduct.getByRole('button', { name: /remove/i }).click()
    await expect.element(firstProduct.getByRole('button', { name: /add/i })).toBeVisible()
  })
})

test.describe('Add to Cart from Product Detail', () => {
  test('should add product to cart from product detail page', async () => {
    await page.goto('http://localhost:8080/')

    // Navigate to product detail
    await page.waitForSelector('[data-testid="product-card"]', { timeout: 10000 })
    const firstProduct = page.getByTestId('product-card').first()
    await firstProduct.click()

    await page.waitForURL(/\/products\/.*/, { timeout: 5000 })

    // Add to cart from detail page
    const addButton = page.getByRole('button', { name: /add to cart/i })
    await addButton.click()

    // Button should change state
    const removeButton = page.getByRole('button', { name: /remove/i })
    await expect.element(removeButton).toBeVisible()
  })

  test('should reflect cart state when navigating back to list', async () => {
    await page.goto('http://localhost:8080/')

    // Add product from list
    await page.waitForSelector('[data-testid="product-card"]', { timeout: 10000 })
    const firstProduct = page.getByTestId('product-card').first()
    await firstProduct.getByRole('button', { name: /add/i }).click()

    // Navigate to detail page
    await firstProduct.click()
    await page.waitForURL(/\/products\/.*/, { timeout: 5000 })

    // Should show "Remove" button
    const removeButton = page.getByRole('button', { name: /remove/i })
    await expect.element(removeButton).toBeVisible()

    // Navigate back
    await page.goBack()

    // Should still show "Remove" button in list
    await page.waitForSelector('[data-testid="product-card"]', { timeout: 10000 })
    const productInList = page.getByTestId('product-card').first()
    await expect.element(productInList.getByRole('button', { name: /remove/i })).toBeVisible()
  })
})
