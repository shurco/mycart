import { CURRENCIES } from '$lib/config/currencies'
import type { TruncationSettings, NumberFormatSettings } from '$lib/types/models'
import { getCurrencyPattern, getUnitLabel, selectFlexibleUnit } from '$lib/config/currencyUnits'

// Apply decimal precision and trailing zeros rules
function applyDecimalPrecision(
  value: number,
  settings?: NumberFormatSettings
): string {
  // Default: precision=2, trailing_zeros=true (backward compatible)
  const precision = settings?.decimal_precision ?? 2
  const showTrailing = settings?.show_trailing_zeros ?? true

  // Step 1: Round to desired precision
  const rounded = value.toFixed(precision)

  // Step 2: Remove trailing zeros if disabled
  if (!showTrailing) {
    // parseFloat removes trailing zeros: "1.00" → "1", "1.20" → "1.2"
    return parseFloat(rounded).toString()
  }

  // Keep trailing zeros: "1.00", "1.20", "1.23"
  return rounded
}

export function formatCurrency(amount: number, currencyCode: string, numberFormat?: NumberFormatSettings): string {
  const currency = CURRENCIES.find(c => c.code === currencyCode)

  if (!currency) {
    // Fallback for unknown currencies
    return `${amount} ${currencyCode}`
  }

  const precision = numberFormat?.decimal_precision ?? 2
  const showTrailing = numberFormat?.show_trailing_zeros ?? true

  const formatted = amount.toLocaleString('en-US', {
    minimumFractionDigits: showTrailing ? precision : 0,
    maximumFractionDigits: precision
  })

  return `${currency.symbol}${formatted}`
}

export function formatCurrencyWithTruncation(
  amount: number,
  currencyCode: string,
  context: 'admin' | 'storefront',
  truncationSettings?: TruncationSettings,
  locale?: string,
  numberFormat?: NumberFormatSettings
): string {
  // Default to 'none' mode if settings missing
  if (!truncationSettings || !truncationSettings[context]) {
    return formatCurrency(amount / 100, currencyCode, numberFormat)
  }

  const settings = truncationSettings[context][currencyCode]
  if (!settings || settings.mode === 'none') {
    return formatCurrency(amount / 100, currencyCode, numberFormat)
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

    if (unit && amount >= unit.value * 100) {
      const divided = (amount / 100) / unit.value
      const formatted = applyDecimalPrecision(divided, numberFormat)
      const unitLabel = getUnitLabel(unit.value, currencyCode, currentLocale)
      return `${currency.symbol}${formatted}${unitLabel}`
    }

    // Amount below unit threshold - show full amount
    return formatCurrency(amount / 100, currencyCode, numberFormat)
  }

  // Flexible mode: auto-select best unit
  if (settings.mode === 'flexible') {
    const unit = selectFlexibleUnit(amount / 100, currencyCode)

    if (unit) {
      const divided = (amount / 100) / unit.value
      const formatted = applyDecimalPrecision(divided, numberFormat)
      const unitLabel = getUnitLabel(unit.value, currencyCode, currentLocale)
      return `${currency.symbol}${formatted}${unitLabel}`
    }

    // No unit found - show full amount
    return formatCurrency(amount / 100, currencyCode, numberFormat)
  }

  // Fallback to full formatting
  return formatCurrency(amount / 100, currencyCode, numberFormat)
}
