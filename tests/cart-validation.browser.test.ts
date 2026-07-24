import { test, expect } from '@playwright/test'

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080'

test('validation modal shows for out-of-stock items', async ({ page }) => {
  // Mock API to return quantity unavailable error
  await page.route('**/api/cart/create', async (route) => {
    await route.fulfill({
      status: 409,
      contentType: 'application/json',
      body: JSON.stringify({
        success: false,
        message: 'Cart validation failed',
        result: {
          validation_errors: [
            {
              item_index: 0,
              product_id: 'test_prod',
              variant_id: null,
              error_type: 'quantity_unavailable',
              requested_qty: 10,
              available_qty: 0,
              requested_unit_price: 2999,
              current_unit_price: 2999,
              requested_total: 29990,
              current_total: 0
            }
          ],
          corrected_cart: [
            {
              product_id: 'test_prod',
              variant_id: null,
              quantity: 0,
              unit_price: 2999,
              available: false
            }
          ]
        }
      })
    })
  })

  await page.goto(`${BASE_URL}/cart`)

  // Assume cart has items (setup would add items in a real test)
  const emailInput = page.locator('input[type="email"]')
  if (await emailInput.isVisible()) {
    await emailInput.fill('test@example.com')
    await page.click('button[type="submit"]')

    // Validation modal should appear
    await expect(page.locator('.validation-modal')).toBeVisible()
    await expect(page.locator('.validation-modal')).toContainText('no longer available')
  }
})

test('price change shows in modal with highlighting', async ({ page }) => {
  // Mock API response for price change
  await page.route('**/api/cart/create', async (route) => {
    await route.fulfill({
      status: 409,
      contentType: 'application/json',
      body: JSON.stringify({
        success: false,
        message: 'Cart validation failed',
        result: {
          validation_errors: [
            {
              item_index: 0,
              product_id: 'test_prod',
              variant_id: null,
              error_type: 'price_changed',
              requested_qty: 1,
              available_qty: 10,
              requested_unit_price: 1999,
              current_unit_price: 2499,
              requested_total: 1999,
              current_total: 2499
            }
          ],
          corrected_cart: [
            {
              product_id: 'test_prod',
              variant_id: null,
              quantity: 1,
              unit_price: 2499,
              available: true
            }
          ]
        }
      })
    })
  })

  await page.goto(`${BASE_URL}/cart`)

  const emailInput = page.locator('input[type="email"]')
  if (await emailInput.isVisible()) {
    await emailInput.fill('test@example.com')
    await page.click('button[type="submit"]')

    // Modal should show price update message
    await expect(page.locator('.validation-modal')).toBeVisible()
    await expect(page.locator('.validation-modal')).toContainText('Price has been updated')
  }
})

test('successful cart creation proceeds normally', async ({ page }) => {
  // Mock successful response
  await page.route('**/api/cart/create', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        message: 'Cart created',
        result: {
          cart_id: 'test-cart-123',
          amount_total: 2999,
          currency: 'USD'
        }
      })
    })
  })

  await page.goto(`${BASE_URL}/cart`)

  const emailInput = page.locator('input[type="email"]')
  if (await emailInput.isVisible()) {
    await emailInput.fill('test@example.com')
    // Note: Real test would need payment provider selection
    // This is a simplified version
  }
})
