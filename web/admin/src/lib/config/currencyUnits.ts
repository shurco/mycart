// Currency unit definitions for price truncation
export interface UnitDefinition {
  value: number                      // numeric value (1000, 10000, etc.)
  keys: Record<string, string>       // localized unit labels { en: 'K', ko: '천' }
}

export interface CurrencyPattern {
  type: 'usd' | 'krw'
  units: UnitDefinition[]            // sorted largest to smallest
}

// USD pattern: K (1000), M (1000000), B (1000000000)
// Used by: USD, GBP, EUR, CAD, AUD, CHF, CNY, SEK
export const USD_PATTERN: CurrencyPattern = {
  type: 'usd',
  units: [
    { value: 1000000000, keys: { en: 'B' } },
    { value: 1000000, keys: { en: 'M' } },
    { value: 1000, keys: { en: 'K' } }
  ]
}

// KRW pattern: 천, 만, 십만, 백만, 천만 (Korean)
// Used by: KRW
export const KRW_PATTERN: CurrencyPattern = {
  type: 'krw',
  units: [
    { value: 10000000, keys: { en: '10M', ko: '천만' } },
    { value: 1000000, keys: { en: '1M', ko: '백만' } },
    { value: 100000, keys: { en: '100K', ko: '십만' } },
    { value: 10000, keys: { en: '10K', ko: '만' } },
    { value: 1000, keys: { en: '1K', ko: '천' } }
  ]
}

// JPY pattern: 千, 万, 十万, 百万, 千万 (Japanese)
// Used by: JPY
export const JPY_PATTERN: CurrencyPattern = {
  type: 'krw',
  units: [
    { value: 10000000, keys: { en: '10M', ja: '千万' } },
    { value: 1000000, keys: { en: '1M', ja: '百万' } },
    { value: 100000, keys: { en: '100K', ja: '十万' } },
    { value: 10000, keys: { en: '10K', ja: '万' } },
    { value: 1000, keys: { en: '1K', ja: '千' } }
  ]
}

// Map currency codes to patterns
export const CURRENCY_PATTERNS: Record<string, CurrencyPattern> = {
  'USD': USD_PATTERN,
  'GBP': USD_PATTERN,
  'EUR': USD_PATTERN,
  'CAD': USD_PATTERN,
  'AUD': USD_PATTERN,
  'CHF': USD_PATTERN,
  'CNY': USD_PATTERN,
  'SEK': USD_PATTERN,
  'KRW': KRW_PATTERN,
  'JPY': JPY_PATTERN
}

// Get pattern for a currency (fallback to USD pattern if not found)
export function getCurrencyPattern(currencyCode: string): CurrencyPattern {
  return CURRENCY_PATTERNS[currencyCode] || USD_PATTERN
}

// Get localized unit label for a unit value
export function getUnitLabel(unitValue: number, currencyCode: string, locale: string): string {
  const pattern = getCurrencyPattern(currencyCode)
  const unit = pattern.units.find(u => u.value === unitValue)
  if (!unit) return ''

  // For KRW/JPY, always use their native locale regardless of UI language
  const effectiveLocale = currencyCode === 'KRW' ? 'ko'
                        : currencyCode === 'JPY' ? 'ja'
                        : locale

  // Return localized label, fallback to English
  return unit.keys[effectiveLocale] || unit.keys['en'] || ''
}

// Find appropriate unit for flexible mode (returns largest unit that fits)
export function selectFlexibleUnit(amount: number, currencyCode: string): UnitDefinition | null {
  const pattern = getCurrencyPattern(currencyCode)

  // Find largest unit where amount >= unit.value
  for (const unit of pattern.units) {
    if (amount >= unit.value) {
      return unit
    }
  }

  return null // amount too small for any unit
}
