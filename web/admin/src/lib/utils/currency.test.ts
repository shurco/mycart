import { describe, it, expect } from 'vitest'
import { formatCurrencyWithTruncation } from './currency'

describe('formatCurrencyWithTruncation', () => {
  describe('None mode', () => {
    it('should format USD with full precision when mode is none', () => {
      const result = formatCurrencyWithTruncation(
        135200, // $1352.00
        'USD',
        'admin',
        {
          admin: { USD: { mode: 'none' } },
          storefront: {}
        },
        'en'
      )
      expect(result).toBe('$1352.00')
    })

    it('should format KRW with full precision when mode is none', () => {
      const result = formatCurrencyWithTruncation(
        1352,
        'KRW',
        'admin',
        {
          admin: { KRW: { mode: 'none' } },
          storefront: {}
        },
        'ko'
      )
      expect(result).toBe('₩1,352')
    })
  })

  describe('Fixed mode', () => {
    it('should truncate USD to K when amount >= 1000', () => {
      const result = formatCurrencyWithTruncation(
        135200, // $1352.00 → $1.35K
        'USD',
        'storefront',
        {
          admin: {},
          storefront: { USD: { mode: 'fixed', fixed_unit: 'K' } }
        },
        'en'
      )
      expect(result).toBe('$1.35K')
    })

    it('should show full amount when below fixed unit threshold', () => {
      const result = formatCurrencyWithTruncation(
        50000, // $500.00 with K unit → show full $500.00
        'USD',
        'storefront',
        {
          admin: {},
          storefront: { USD: { mode: 'fixed', fixed_unit: 'K' } }
        },
        'en'
      )
      expect(result).toBe('$500.00')
    })

    it('should truncate KRW to 만 with Korean locale', () => {
      const result = formatCurrencyWithTruncation(
        1000000, // ₩10,000.00 → ₩1.00만 (10000/만)
        'KRW',
        'storefront',
        {
          admin: {},
          storefront: { KRW: { mode: 'fixed', fixed_unit: '만' } }
        },
        'ko'
      )
      expect(result).toBe('₩1.00만')
    })

    it('should truncate KRW to 10K with English locale', () => {
      const result = formatCurrencyWithTruncation(
        1000000, // ₩10,000.00 → ₩1.0010K
        'KRW',
        'storefront',
        {
          admin: {},
          storefront: { KRW: { mode: 'fixed', fixed_unit: '만' } }
        },
        'en'
      )
      expect(result).toBe('₩1.0010K')
    })
  })

  describe('Flexible mode', () => {
    it('should not truncate USD below 1K threshold', () => {
      const result = formatCurrencyWithTruncation(
        99900, // $999.00 → show full
        'USD',
        'storefront',
        {
          admin: {},
          storefront: { USD: { mode: 'flexible' } }
        },
        'en'
      )
      expect(result).toBe('$999.00')
    })

    it('should auto-select K for USD amounts >= 1000', () => {
      const result = formatCurrencyWithTruncation(
        135200, // $1352.00 → $1.35K
        'USD',
        'storefront',
        {
          admin: {},
          storefront: { USD: { mode: 'flexible' } }
        },
        'en'
      )
      expect(result).toBe('$1.35K')
    })

    it('should auto-select M for USD amounts >= 1000000', () => {
      const result = formatCurrencyWithTruncation(
        156200000, // $1,562,000.00 → $1.56M
        'USD',
        'storefront',
        {
          admin: {},
          storefront: { USD: { mode: 'flexible' } }
        },
        'en'
      )
      expect(result).toBe('$1.56M')
    })

    it('should auto-select appropriate KRW unit with Korean locale', () => {
      const result = formatCurrencyWithTruncation(
        1500000000, // ₩15,000,000.00 → ₩1.50천만
        'KRW',
        'storefront',
        {
          admin: {},
          storefront: { KRW: { mode: 'flexible' } }
        },
        'ko'
      )
      expect(result).toBe('₩1.50천만')
    })

    it('should auto-select appropriate KRW unit with English locale', () => {
      const result = formatCurrencyWithTruncation(
        1500000000, // ₩15,000,000.00 → ₩15.00M
        'KRW',
        'storefront',
        {
          admin: {},
          storefront: { KRW: { mode: 'flexible' } }
        },
        'en'
      )
      expect(result).toBe('₩15.00M')
    })
  })

  describe('Edge cases', () => {
    it('should handle zero amount', () => {
      const result = formatCurrencyWithTruncation(
        0,
        'USD',
        'storefront',
        {
          admin: {},
          storefront: { USD: { mode: 'flexible' } }
        },
        'en'
      )
      expect(result).toBe('$0.00')
    })

    it('should fallback to none mode when settings undefined', () => {
      const result = formatCurrencyWithTruncation(
        135200,
        'USD',
        'storefront',
        undefined,
        'en'
      )
      expect(result).toBe('$1352.00')
    })

    it('should fallback to none mode when currency missing from settings', () => {
      const result = formatCurrencyWithTruncation(
        135200,
        'GBP',
        'storefront',
        {
          admin: {},
          storefront: { USD: { mode: 'flexible' } } // GBP not configured
        },
        'en'
      )
      expect(result).toBe('£1352.00')
    })

    it('should handle very large amounts', () => {
      const result = formatCurrencyWithTruncation(
        99999999990000, // $999,999,999.00 → $9999999.99M
        'USD',
        'storefront',
        {
          admin: {},
          storefront: { USD: { mode: 'flexible' } }
        },
        'en'
      )
      expect(result).toBe('$9999999.99M')
    })
  })
})
