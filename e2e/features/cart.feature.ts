import { Page } from 'patchright'
import { expect } from 'patchright/test'

/**
 * Feature Object for Shopping Cart Page (Storefront)
 * Handles cart viewing and product removal
 */
export class CartFeature {
  constructor(private page: Page) {}

  async goto() {
    await this.page.goto('/cart')
  }

  async waitForCart() {
    // Wait for the cart page to fully load - wait for either items or empty message
    await this.page.waitForSelector('[data-testid="cart-item"], text=/cart is empty/i', { timeout: 10000 })
  }

  async verifyPageLoaded() {
    await expect(this.page).toHaveURL('/cart')
    await this.waitForCart()
  }

  async getCartItemCount() {
    const items = await this.page.locator('[data-testid="cart-item"]').count()
    return items
  }

  async verifyCartIsEmpty() {
    // Use getByRole to target only the h1 heading, not the paragraph text
    const emptyMessage = this.page.getByRole('heading', { name: /cart is empty/i })
    await expect(emptyMessage).toBeVisible()
  }

  async verifyCartHasItems(count: number) {
    // Wait a moment for cart to load from localStorage
    await this.page.waitForTimeout(1000)
    const items = await this.getCartItemCount()
    expect(items).toBe(count)
  }

  async getItemNameByIndex(index: number) {
    const item = this.page.locator('[data-testid="cart-item"]').nth(index)
    const name = await item.locator('[data-testid="item-name"]').textContent()
    return name?.trim() || ''
  }

  async removeItemByIndex(index: number) {
    const item = this.page.locator('[data-testid="cart-item"]').nth(index)
    const removeButton = item.locator('button:has-text("REMOVE"), button[aria-label*="remove" i]')
    await removeButton.click()
  }

  async verifyItemRemoved(productName: string) {
    const item = this.page.locator(`[data-testid="cart-item"]:has-text("${productName}")`)
    await expect(item).not.toBeVisible()
  }

  async getTotalPrice() {
    const total = await this.page.locator('[data-testid="cart-total"]').textContent()
    return total?.trim() || ''
  }
}
