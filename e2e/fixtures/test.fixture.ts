import { test as base } from 'patchright/test'
import { ProductListFeature } from '../features/product-list.feature'
import { ProductDetailFeature } from '../features/product-detail.feature'
import { CartFeature } from '../features/cart.feature'
import { AdminProductsFeature } from '../features/admin-products.feature'

type Fixtures = {
  productList: ProductListFeature
  productDetail: ProductDetailFeature
  cart: CartFeature
  adminProducts: AdminProductsFeature
}

export const test = base.extend<Fixtures>({
  productList: async ({ page }, use) => {
    const productList = new ProductListFeature(page)
    await use(productList)
  },
  productDetail: async ({ page }, use) => {
    const productDetail = new ProductDetailFeature(page)
    await use(productDetail)
  },
  cart: async ({ page }, use) => {
    const cart = new CartFeature(page)
    await use(cart)
  },
  adminProducts: async ({ page }, use) => {
    const adminProducts = new AdminProductsFeature(page)
    await use(adminProducts)
  },
})

export { expect } from 'patchright/test'
