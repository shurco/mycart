import { expect, test } from 'vitest'
import { page } from '@vitest/browser/context'

// TDD RED Phase: Write failing tests for admin page views
test.describe('Admin Page Views', () => {
  // Note: These tests assume the admin is already authenticated
  // or that the test environment has authentication disabled

  test('should display admin dashboard/home page', async () => {
    await page.goto('http://localhost:8080/admin')

    // Check if dashboard loads (might redirect to signin if not authenticated)
    const currentUrl = page.url()

    if (currentUrl.includes('/signin')) {
      // Skip test if authentication is required
      test.skip()
    } else {
      // Dashboard should have navigation or main content
      const mainContent = page.locator('main, [role="main"]')
      await expect.element(mainContent).toBeVisible()
    }
  })

  test('should display products page', async () => {
    await page.goto('http://localhost:8080/admin/products')

    // Check if redirected to signin
    if (page.url().includes('/signin')) {
      test.skip()
    }

    // Products page should have product list or add button
    const productsHeading = page.getByRole('heading', { name: /products/i })
    await expect.element(productsHeading).toBeVisible()
  })

  test('should display carts page', async () => {
    await page.goto('http://localhost:8080/admin/carts')

    if (page.url().includes('/signin')) {
      test.skip()
    }

    // Carts page should have heading
    const cartsHeading = page.getByRole('heading', { name: /carts|orders/i })
    await expect.element(cartsHeading).toBeVisible()
  })

  test('should display pages management page', async () => {
    await page.goto('http://localhost:8080/admin/pages')

    if (page.url().includes('/signin')) {
      test.skip()
    }

    // Pages management should have heading
    const pagesHeading = page.getByRole('heading', { name: /pages/i })
    await expect.element(pagesHeading).toBeVisible()
  })

  test('should navigate between admin pages', async () => {
    await page.goto('http://localhost:8080/admin/products')

    if (page.url().includes('/signin')) {
      test.skip()
    }

    // Find navigation link to carts
    const cartsLink = page.getByRole('link', { name: /carts/i })
    await cartsLink.click()

    // Should navigate to carts page
    await page.waitForURL(/\/admin\/carts/, { timeout: 5000 })
    expect(page.url()).toContain('/admin/carts')
  })
})
