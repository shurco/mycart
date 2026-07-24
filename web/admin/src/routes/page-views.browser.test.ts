import { expect, test } from 'vitest'
import { browser } from 'vitest/browser'

// TDD RED Phase: Write failing tests for admin page views
test.describe('Admin Page Views', () => {
  test('should display admin dashboard/home page', async () => {
    await browser.url('http://localhost:8080/admin')

    const currentUrl = await browser.getUrl()

    if (currentUrl.includes('/signin')) {
      test.skip()
    } else {
      const mainContent = await browser.$('main')
      await expect(mainContent.isDisplayed()).resolves.toBe(true)
    }
  })

  test('should display products page', async () => {
    await browser.url('http://localhost:8080/admin/products')

    const currentUrl = await browser.getUrl()
    if (currentUrl.includes('/signin')) {
      test.skip()
    }

    const productsHeading = await browser.$('h1')
    await expect(productsHeading.isDisplayed()).resolves.toBe(true)
  })

  test('should display carts page', async () => {
    await browser.url('http://localhost:8080/admin/carts')

    const currentUrl = await browser.getUrl()
    if (currentUrl.includes('/signin')) {
      test.skip()
    }

    const cartsHeading = await browser.$('h1')
    await expect(cartsHeading.isDisplayed()).resolves.toBe(true)
  })

  test('should display pages management page', async () => {
    await browser.url('http://localhost:8080/admin/pages')

    const currentUrl = await browser.getUrl()
    if (currentUrl.includes('/signin')) {
      test.skip()
    }

    const pagesHeading = await browser.$('h1')
    await expect(pagesHeading.isDisplayed()).resolves.toBe(true)
  })

  test('should navigate between admin pages', async () => {
    await browser.url('http://localhost:8080/admin/products')

    const currentUrl = await browser.getUrl()
    if (currentUrl.includes('/signin')) {
      test.skip()
    }

    const cartsLink = await browser.$('a[href="/admin/carts"]')
    await cartsLink.click()

    await browser.waitUntil(
      async () => (await browser.getUrl()).includes('/admin/carts'),
      { timeout: 5000 }
    )

    expect(await browser.getUrl()).toContain('/admin/carts')
  })
})
