// web/site/src/lib/utils/currency.ts
import { CURRENCIES } from '$lib/config/currencies'

export function formatCurrency(amount: number, currencyCode: string): string {
  const currency = CURRENCIES.find(c => c.code === currencyCode)
  
  if (!currency) {
    // Fallback for unknown currencies
    return `${amount} ${currencyCode}`
  }

  const formatted = currency.decimals === 0
    ? Math.round(amount).toLocaleString()
    : amount.toFixed(currency.decimals)

  return `${currency.symbol}${formatted}`
}
