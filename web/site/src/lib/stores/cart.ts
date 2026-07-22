import { writable } from 'svelte/store'
import type { CartItem } from '$lib/types/models'
import { isBrowser, getLocalStorage, setLocalStorage } from '$lib/utils/browser'

const CART_STORAGE_KEY = 'cart'

function createCartStore() {
  const loadFromStorage = (): CartItem[] => {
    if (!isBrowser()) return []

    try {
      const stored = getLocalStorage(CART_STORAGE_KEY)
      if (!stored) return []

      const items = JSON.parse(stored)

      // Migration: add quantity=1 to items without quantity field
      return items.map((item: any) => ({
        ...item,
        quantity: item.quantity || 1
      }))
    } catch {
      return []
    }
  }

  const saveToStorage = (items: CartItem[]) => {
    if (!isBrowser()) return

    setLocalStorage(CART_STORAGE_KEY, JSON.stringify(items))
  }

  const { subscribe, set, update } = writable<CartItem[]>(loadFromStorage())

  return {
    subscribe,
    add: (item: CartItem) => {
      update((items) => {
        // Check if this exact item (product + variant) already exists
        const existing = items.find((i) => {
          if (item.variant_id) {
            return i.id === item.id && i.variant_id === item.variant_id
          }
          return i.id === item.id && !i.variant_id
        })

        if (existing) {
          // Accumulate quantity instead of ignoring
          const newItems = items.map(i =>
            i === existing
              ? { ...i, quantity: i.quantity + (item.quantity || 1) }
              : i
          )
          saveToStorage(newItems)
          return newItems
        }

        const newItems = [...items, { ...item, quantity: item.quantity || 1 }]
        saveToStorage(newItems)
        return newItems
      })
    },
    remove: (id: string) => {
      update((items) => {
        const newItems = items.filter((item) => item.id !== id)
        saveToStorage(newItems)
        return newItems
      })
    },
    removeVariant: (productId: string, variantId: string) => {
      update((items) => {
        const newItems = items.filter((item) => !(item.id === productId && item.variant_id === variantId))
        saveToStorage(newItems)
        return newItems
      })
    },
    updateQuantity: (productId: string, variantId: string | undefined, quantity: number) => {
      update((items) => {
        const newItems = items.map(item => {
          const matches = variantId
            ? (item.id === productId && item.variant_id === variantId)
            : (item.id === productId && !item.variant_id)

          if (matches) {
            return { ...item, quantity: Math.max(1, quantity) }
          }
          return item
        })
        saveToStorage(newItems)
        return newItems
      })
    },
    incrementQuantity: (productId: string, variantId: string | undefined) => {
      update((items) => {
        const newItems = items.map(item => {
          const matches = variantId
            ? (item.id === productId && item.variant_id === variantId)
            : (item.id === productId && !item.variant_id)

          if (matches) {
            return { ...item, quantity: item.quantity + 1 }
          }
          return item
        })
        saveToStorage(newItems)
        return newItems
      })
    },
    decrementQuantity: (productId: string, variantId: string | undefined) => {
      update((items) => {
        const newItems = items.map(item => {
          const matches = variantId
            ? (item.id === productId && item.variant_id === variantId)
            : (item.id === productId && !item.variant_id)

          if (matches) {
            return { ...item, quantity: Math.max(1, item.quantity - 1) }
          }
          return item
        })
        saveToStorage(newItems)
        return newItems
      })
    },
    clear: () => {
      set([])
      saveToStorage([])
    },
    reload: () => {
      set(loadFromStorage())
    }
  }
}

export const cartStore = createCartStore()
