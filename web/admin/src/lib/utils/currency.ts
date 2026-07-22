import { CURRENCIES } from '$lib/config/currencies'
import type { TruncationSettings } from '$lib/types/models'
import { getCurrencyPattern, getUnitLabel, selectFlexibleUnit } from '$lib/config/currencyUnits'

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

export function formatCurrencyWithTruncation(
  amount: number,
  currencyCode: string,
  context: 'admin' | 'storefront',
  truncationSettings?: TruncationSettings,
  locale?: string
): string {
  // Default to 'none' mode if settings missing
  if (!truncationSettings || !truncationSettings[context]) {
    return formatCurrency(amount / 100, currencyCode)
  }

  const settings = truncationSettings[context][currencyCode]
  if (!settings || settings.mode === 'none') {
    return formatCurrency(amount / 100, currencyCode)
  }

  const currency = CURRENCIES.find(c => c.code === currencyCode)
  if (!currency) {
    return `${amount / 100} ${currencyCode}`
  }

  const pattern = getCurrencyPattern(currencyCode)
  const currentLocale = locale || 'en'

  // Fixed mode: use specified unit
  if (settings.mode === 'fixed' && settings.fixed_unit) {
    // Find unit by matching localized labels
    const unit = pattern.units.find(u => {
      const label = getUnitLabel(u.value, currencyCode, currentLocale)
      return label === settings.fixed_unit ||
             Object.values(u.keys).includes(settings.fixed_unit)
    })

    if (unit && amount >= unit.value) {
      const divided = (amount / 100) / (unit.value / 100)
      const formatted = divided.toFixed(2)
      const unitLabel = getUnitLabel(unit.value, currencyCode, currentLocale)
      return `${currency.symbol}${formatted}${unitLabel}`
    }

    // Amount below unit threshold - show full amount
    return formatCurrency(amount / 100, currencyCode)
  }

  // Flexible mode: auto-select best unit
  if (settings.mode === 'flexible') {
    const unit = selectFlexibleUnit(amount / 100, currencyCode)

    if (unit) {
      const divided = (amount / 100) / (unit.value / 100)
      const formatted = divided.toFixed(2)
      const unitLabel = getUnitLabel(unit.value, currencyCode, currentLocale)
      return `${currency.symbol}${formatted}${unitLabel}`
    }

    // No unit found - show full amount
    return formatCurrency(amount / 100, currencyCode)
  }

  // Fallback to full formatting
  return formatCurrency(amount / 100, currencyCode)
}
