import { describe, it, expect, vi } from 'vitest'
import { writable } from 'svelte/store'
import { formatCurrency } from './currency'

// Mock the locale store
vi.mock('$lib/i18n', () => ({
  locale: writable('en')
}))

describe('formatCurrency', () => {
  it('returns "free" for zero amount', () => {
    expect(formatCurrency(0, 'USD')).toBe('free')
    expect(formatCurrency(0, 'KRW')).toBe('free')
  })

  it('formats KRW for English locale with ₩ prefix', async () => {
    const { locale } = await import('$lib/i18n')
    locale.set('en')
    expect(formatCurrency(300000, 'KRW')).toBe('₩3,000')
  })

  it('formats KRW for Korean locale with 원 suffix', async () => {
    const { locale } = await import('$lib/i18n')
    locale.set('ko')
    expect(formatCurrency(300000, 'KRW')).toBe('3,000원')
  })

  it('formats KRW with no decimals', async () => {
    const { locale } = await import('$lib/i18n')
    locale.set('en')
    expect(formatCurrency(350050, 'KRW')).toBe('₩3,501')
  })

  it('formats USD with decimals', () => {
    expect(formatCurrency(3000, 'USD')).toBe('30.00 USD')
  })

  it('formats EUR with decimals', () => {
    expect(formatCurrency(5000, 'EUR')).toBe('50.00 EUR')
  })
})
