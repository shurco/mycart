import { loadData } from '$lib/utils/apiHelpers'
import { sortByDate, showMessage } from '$lib/utils'
import { apiUpdate } from '$lib/utils/api'
import { DEFAULT_PAGE_SIZE } from '$lib/constants/pagination'
import type { Product } from '$lib/types/models'
import { t } from '$lib/i18n'

interface ProductsResponse {
  products: Product[]
  currency: string
  total: number
}

/** The paged product list behind the table: what was loaded and the row actions on it. */
export class ProductsList {
  products = $state<Product[]>([])
  currency = $state('')
  loading = $state(true)
  currentPage = $state(1)
  limit = $state(DEFAULT_PAGE_SIZE)
  total = $state(0)

  constructor() {
    // DEBUG: Watch for products array changes
    $effect(() => {
      const ids = this.products.map(p => p.id)
      const uniqueIds = new Set(ids)
      if (ids.length !== uniqueIds.size) {
        console.error('[EFFECT] Duplicate IDs detected in products array!')
        console.error('IDs:', ids)
        console.error('Duplicates:', ids.filter((id, i) => ids.indexOf(id) !== i))
      }
    })
  }

  load = async (page = this.currentPage) => {
    console.log(`[loadProducts] Starting - page ${page}`)
    this.loading = true
    this.currentPage = page
    const result = await loadData<ProductsResponse>(
      `/api/_/products?page=${page}&limit=${this.limit}`,
      t('products.failedToLoad')
    )

    console.log('[loadProducts] API response:', result)

    if (result) {
      this.products = sortByDate(result.products || [])
      this.currency = result.currency || ''
      this.total = result.total || 0
    }
    this.loading = false
    console.log('[loadProducts] Finished')
  }

  toggleActive = async (product: Product, index: number) => {
    const originalActive = product.active
    const newActive = !product.active

    // Optimistic update - update directly instead of map
    this.products[index] = { ...this.products[index], active: newActive }

    try {
      const res = await apiUpdate(`/api/_/products/${product.id}/active`, {})
      if (!res.success) {
        // Revert on failure
        this.products[index] = { ...this.products[index], active: originalActive }
        showMessage(res.message || t('products.failedToUpdateProduct'), 'connextError')
      } else {
        showMessage(t('products.productStatusUpdated') || res.message, 'connextSuccess')
      }
    } catch (error) {
      // Revert on error
      this.products[index] = { ...this.products[index], active: originalActive }
      showMessage(t('common.networkError'), 'connextError')
    }
  }
}
