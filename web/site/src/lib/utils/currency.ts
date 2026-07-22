import { get } from 'svelte/store'
import { locale } from '$lib/i18n'

export function formatCurrency(amount: number, currency: string): string {
  if (!amount || amount === 0) return 'free'

  const value = amount / 100 // Convert from cents

  // KRW specific formatting
  if (currency === 'KRW') {
    const currentLocale = get(locale)
    const rounded = Math.round(value) // No decimals for KRW
    const formatted = rounded.toLocaleString()

    // Korean UI: "3,000원"
    if (currentLocale === 'ko') {
      return `${formatted}원`
    }
    // English UI: "₩3,000"
    return `₩${formatted}`
  }

  // Default formatting for other currencies
  return `${value.toFixed(2)} ${currency}`
}
