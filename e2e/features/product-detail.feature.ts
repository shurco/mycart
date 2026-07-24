import { Page } from 'patchright'
import { expect } from 'patchright/test'

/**
 * Feature Object for Product Detail Page (Storefront)
 * Handles single product view and add-to-cart from detail page
 */
export class ProductDetailFeature {
  constructor(private page: Page) {}

  async gotoProduct(slug: string) {
    await this.page.goto(`/products/${slug}`)
  }

  async waitForProductDetails() {
    await this.page.waitForSelector('h1', { timeout: 10000 })
  }

  async verifyPageLoaded(productName: string) {
    await this.waitForProductDetails()
    const title = await this.page.locator('h1').textContent()
    expect(title).toContain(productName)
  }

  async getProductName() {
    const name = await this.page.locator('h1').textContent()
    return name?.trim() || ''
  }

  async addToCart() {
    // Look for the green add button (bg-green-500) - the main CTA button on page
    const addButton = this.page.locator('button.bg-green-500').first()
    await addButton.waitFor({ state: 'visible', timeout: 5000 })
    await addButton.click()
    // Wait for button to change to red (verifies item was added)
    // Use last() to get the product button, not the header cart button
    const removeButton = this.page.locator('button.bg-red-500').last()
    await removeButton.waitFor({ state: 'visible', timeout: 5000 })
  }

  async verifyInCart() {
    // Look for the red remove button (bg-red-500) on the product (not header)
    const removeButton = this.page.locator('button.bg-red-500').last()
    await expect(removeButton).toBeVisible({ timeout: 5000 })
  }

  async getProductPrice() {
    // Price is in the large text near the add/remove button
    const priceText = await this.page.locator('text=/\\$|₩|£/').first().textContent()
    return priceText?.trim() || ''
  }
}
