import { describe, it, expect, beforeEach } from 'vitest'
import { get } from 'svelte/store'
import { cartStore } from './cart'
import type { CartItem } from '$lib/types/models'

describe('Cart Store', () => {
  beforeEach(() => {
    // Clear cart and localStorage before each test
    cartStore.clear()
    localStorage.clear()
  })

  describe('add', () => {
    it('should add item to empty cart', () => {
      const item: CartItem = {
        id: '1',
        name: 'Test Product',
        slug: 'test-product',
        amount: 1000,
        image: null
      }

      cartStore.add(item)

      const cart = get(cartStore)
      expect(cart).toHaveLength(1)
      expect(cart[0]).toEqual(item)
    })

    it('should not add duplicate item', () => {
      const item: CartItem = {
        id: '1',
        name: 'Test Product',
        slug: 'test-product',
        amount: 1000,
        image: null
      }

      cartStore.add(item)
      cartStore.add(item)

      const cart = get(cartStore)
      expect(cart).toHaveLength(1)
    })

    it('should add multiple different items', () => {
      const item1: CartItem = {
        id: '1',
        name: 'Product 1',
        slug: 'product-1',
        amount: 1000,
        image: null
      }

      const item2: CartItem = {
        id: '2',
        name: 'Product 2',
        slug: 'product-2',
        amount: 2000,
        image: null
      }

      cartStore.add(item1)
      cartStore.add(item2)

      const cart = get(cartStore)
      expect(cart).toHaveLength(2)
      expect(cart[0]).toEqual(item1)
      expect(cart[1]).toEqual(item2)
    })
  })

  describe('remove', () => {
    it('should remove item from cart', () => {
      const item: CartItem = {
        id: '1',
        name: 'Test Product',
        slug: 'test-product',
        amount: 1000,
        image: null
      }

      cartStore.add(item)
      cartStore.remove('1')

      const cart = get(cartStore)
      expect(cart).toHaveLength(0)
    })

    it('should remove only specified item', () => {
      const item1: CartItem = {
        id: '1',
        name: 'Product 1',
        slug: 'product-1',
        amount: 1000,
        image: null
      }

      const item2: CartItem = {
        id: '2',
        name: 'Product 2',
        slug: 'product-2',
        amount: 2000,
        image: null
      }

      cartStore.add(item1)
      cartStore.add(item2)
      cartStore.remove('1')

      const cart = get(cartStore)
      expect(cart).toHaveLength(1)
      expect(cart[0]).toEqual(item2)
    })

    it('should handle removing non-existent item', () => {
      const item: CartItem = {
        id: '1',
        name: 'Product 1',
        slug: 'product-1',
        amount: 1000,
        image: null
      }

      cartStore.add(item)
      cartStore.remove('999')

      const cart = get(cartStore)
      expect(cart).toHaveLength(1)
      expect(cart[0]).toEqual(item)
    })
  })

  describe('clear', () => {
    it('should clear all items from cart', () => {
      const item1: CartItem = {
        id: '1',
        name: 'Product 1',
        slug: 'product-1',
        amount: 1000,
        image: null
      }

      const item2: CartItem = {
        id: '2',
        name: 'Product 2',
        slug: 'product-2',
        amount: 2000,
        image: null
      }

      cartStore.add(item1)
      cartStore.add(item2)
      cartStore.clear()

      const cart = get(cartStore)
      expect(cart).toHaveLength(0)
    })

    it('should clear localStorage', () => {
      const item: CartItem = {
        id: '1',
        name: 'Product 1',
        slug: 'product-1',
        amount: 1000,
        image: null
      }

      cartStore.add(item)
      expect(localStorage.getItem('cart')).not.toBeNull()

      cartStore.clear()
      expect(localStorage.getItem('cart')).toBe('[]')
    })
  })

  describe('persistence', () => {
    it('should persist items to localStorage', () => {
      const item: CartItem = {
        id: '1',
        name: 'Test Product',
        slug: 'test-product',
        amount: 1000,
        image: null
      }

      cartStore.add(item)

      const stored = localStorage.getItem('cart')
      expect(stored).not.toBeNull()
      const parsed = JSON.parse(stored!)
      expect(parsed).toHaveLength(1)
      expect(parsed[0]).toEqual(item)
    })
  })
})
