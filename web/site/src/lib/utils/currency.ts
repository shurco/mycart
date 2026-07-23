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

export function formatCurrency(
  amount: number,
  currencyCode: string,
  numberFormat?: NumberFormatSettings,
  symbolMode?: 'currency' | 'language',
  locale?: string
): string {
  const currency = CURRENCIES.find(c => c.code === currencyCode)

  if (!currency) {
    return `${amount} ${currencyCode}`
  }

  const precision = numberFormat?.decimal_precision ?? 2
  const showTrailing = numberFormat?.show_trailing_zeros ?? true

  const formatted = amount.toLocaleString('en-US', {
    minimumFractionDigits: showTrailing ? precision : 0,
    maximumFractionDigits: precision
  })

  if (symbolMode === 'language') {
    const currentLocale = locale || 'en'
    const currencyName = currency.names?.[currentLocale] || currency.names?.en || currency.name
    return `${formatted} ${currencyName}`
  }

  return `${currency.symbol}${formatted}`
}

export function formatCurrencyWithTruncation(
  amount: number,
  currencyCode: string,
  context: 'admin' | 'storefront',
  truncationSettings?: TruncationSettings,
  locale?: string,
  numberFormat?: NumberFormatSettings,
  symbolMode?: 'currency' | 'language'
): string {
  // Default to 'none' mode if settings missing
  if (!truncationSettings || !truncationSettings[context]) {
    return formatCurrency(amount / 100, currencyCode, numberFormat, symbolMode, locale)
  }

  const settings = truncationSettings[context][currencyCode]
  if (!settings || settings.mode === 'none') {
    return formatCurrency(amount / 100, currencyCode, numberFormat, symbolMode, locale)
  }

  const currency = CURRENCIES.find(c => c.code === currencyCode)
  if (!currency) {
    return `${amount / 100} ${currencyCode}`
  }

  const pattern = getCurrencyPattern(currencyCode)
  const currentLocale = locale || 'en'

  // Get currency name for language mode
  const getCurrencyName = () => {
    if (symbolMode === 'language') {
      return currency.names?.[currentLocale] || currency.names?.en || currency.name
    }
    return null
  }

  // Fixed mode: use specified unit
  if (settings.mode === 'fixed' && settings.fixed_unit) {
    const unit = pattern.units.find(u => {
      const label = getUnitLabel(u.value, currencyCode, currentLocale)
      return label === settings.fixed_unit ||
             Object.values(u.keys).includes(settings.fixed_unit)
    })

    if (unit && amount >= unit.value * 100) {
      const divided = (amount / 100) / unit.value
      const formatted = applyDecimalPrecision(divided, numberFormat)
      const unitLabel = getUnitLabel(unit.value, currencyCode, currentLocale)

      const currencyName = getCurrencyName()
      if (currencyName) {
        return `${formatted}${unitLabel} ${currencyName}`
      }
      return `${currency.symbol}${formatted}${unitLabel}`
    }

    return formatCurrency(amount / 100, currencyCode, numberFormat, symbolMode, locale)
  }

  // Flexible mode: auto-select best unit
  if (settings.mode === 'flexible') {
    const unit = selectFlexibleUnit(amount / 100, currencyCode)

    if (unit) {
      const divided = (amount / 100) / unit.value
      const formatted = applyDecimalPrecision(divided, numberFormat)
      const unitLabel = getUnitLabel(unit.value, currencyCode, currentLocale)

      const currencyName = getCurrencyName()
      if (currencyName) {
        return `${formatted}${unitLabel} ${currencyName}`
      }
      return `${currency.symbol}${formatted}${unitLabel}`
    }

    return formatCurrency(amount / 100, currencyCode, numberFormat, symbolMode, locale)
  }

  return formatCurrency(amount / 100, currencyCode, numberFormat, symbolMode, locale)
}
