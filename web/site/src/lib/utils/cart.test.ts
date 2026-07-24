import { describe, it, expect, beforeEach } from 'vitest'
import { get } from 'svelte/store'
import { toggleCartItem } from './cart'
import { cartStore } from '$lib/stores/cart'
import type { Product, CartItem } from '$lib/types/models'

describe('toggleCartItem', () => {
  beforeEach(() => {
    cartStore.clear()
    localStorage.clear()
  })

  describe('adding to cart', () => {
    it('should add product to empty cart', () => {
      const product: Product = {
        id: '1',
        name: 'Test Product',
        slug: 'test-product',
        amount: 1000,
        brief: 'Brief',
        description: 'Description',
        active: true,
        images: [{ name: 'image', ext: 'jpg' }],
        created_at: '2024-01-01',
        updated_at: '2024-01-01'
      }

      const cartItems: CartItem[] = []
      toggleCartItem(product, cartItems)

      const cart = get(cartStore)
      expect(cart).toHaveLength(1)
      expect(cart[0].id).toBe(product.id)
      expect(cart[0].name).toBe(product.name)
      expect(cart[0].slug).toBe(product.slug)
      expect(cart[0].amount).toBe(product.amount)
    })

    it('should preserve product image when adding', () => {
      const product: Product = {
        id: '1',
        name: 'Test Product',
        slug: 'test-product',
        amount: 1000,
        brief: 'Brief',
        description: 'Description',
        active: true,
        images: [{ name: 'test-image', ext: 'png' }],
        created_at: '2024-01-01',
        updated_at: '2024-01-01'
      }

      toggleCartItem(product, [])

      const cart = get(cartStore)
      expect(cart[0].image).toEqual({ name: 'test-image', ext: 'png' })
    })

    it('should handle product with no images', () => {
      const product: Product = {
        id: '1',
        name: 'Test Product',
        slug: 'test-product',
        amount: 1000,
        brief: 'Brief',
        description: 'Description',
        active: true,
        images: [],
        created_at: '2024-01-01',
        updated_at: '2024-01-01'
      }

      toggleCartItem(product, [])

      const cart = get(cartStore)
      expect(cart[0].image).toBeNull()
    })
  })

  describe('removing from cart', () => {
    it('should remove product when already in cart', () => {
      const product: Product = {
        id: '1',
        name: 'Test Product',
        slug: 'test-product',
        amount: 1000,
        brief: 'Brief',
        description: 'Description',
        active: true,
        images: [],
        created_at: '2024-01-01',
        updated_at: '2024-01-01'
      }

      const cartItems: CartItem[] = [
        {
          id: '1',
          name: 'Test Product',
          slug: 'test-product',
          amount: 1000,
          image: null
        }
      ]

      // Add to cart first
      cartStore.add(cartItems[0])

      // Toggle should remove it
      toggleCartItem(product, cartItems)

      const cart = get(cartStore)
      expect(cart).toHaveLength(0)
    })

    it('should only remove specified product', () => {
      const product1: Product = {
        id: '1',
        name: 'Product 1',
        slug: 'product-1',
        amount: 1000,
        brief: 'Brief',
        description: 'Description',
        active: true,
        images: [],
        created_at: '2024-01-01',
        updated_at: '2024-01-01'
      }

      const cartItem1: CartItem = {
        id: '1',
        name: 'Product 1',
        slug: 'product-1',
        amount: 1000,
        image: null
      }

      const cartItem2: CartItem = {
        id: '2',
        name: 'Product 2',
        slug: 'product-2',
        amount: 2000,
        image: null
      }

      cartStore.add(cartItem1)
      cartStore.add(cartItem2)

      const cartItems = [cartItem1, cartItem2]
      toggleCartItem(product1, cartItems)

      const cart = get(cartStore)
      expect(cart).toHaveLength(1)
      expect(cart[0].id).toBe('2')
    })
  })

  describe('toggle behavior', () => {
    it('should add then remove on consecutive toggles', () => {
      const product: Product = {
        id: '1',
        name: 'Test Product',
        slug: 'test-product',
        amount: 1000,
        brief: 'Brief',
        description: 'Description',
        active: true,
        images: [],
        created_at: '2024-01-01',
        updated_at: '2024-01-01'
      }

      // First toggle - add
      toggleCartItem(product, [])
      let cart = get(cartStore)
      expect(cart).toHaveLength(1)

      // Second toggle - remove
      const cartItems: CartItem[] = [cart[0]]
      toggleCartItem(product, cartItems)
      cart = get(cartStore)
      expect(cart).toHaveLength(0)
    })

    it('should handle multiple different products', () => {
      const product1: Product = {
        id: '1',
        name: 'Product 1',
        slug: 'product-1',
        amount: 1000,
        brief: 'Brief',
        description: 'Description',
        active: true,
        images: [],
        created_at: '2024-01-01',
        updated_at: '2024-01-01'
      }

      const product2: Product = {
        id: '2',
        name: 'Product 2',
        slug: 'product-2',
        amount: 2000,
        brief: 'Brief',
        description: 'Description',
        active: true,
        images: [],
        created_at: '2024-01-01',
        updated_at: '2024-01-01'
      }

      toggleCartItem(product1, [])
      let cart = get(cartStore)

      toggleCartItem(product2, cart)
      cart = get(cartStore)

      expect(cart).toHaveLength(2)
    })
  })
})
