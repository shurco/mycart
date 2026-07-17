// web/site/src/lib/config/currencies.ts
export interface CurrencyConfig {
  code: string           // ISO 4217 code
  symbol: string         // Display symbol
  decimals: number       // Decimal places for display
  name: string          // English name
}

export const CURRENCIES: CurrencyConfig[] = [
  { code: 'USD', symbol: '$', decimals: 2, name: 'US Dollar' },
  { code: 'EUR', symbol: '€', decimals: 2, name: 'Euro' },
  { code: 'GBP', symbol: '£', decimals: 2, name: 'British Pound' },
  { code: 'JPY', symbol: '¥', decimals: 0, name: 'Japanese Yen' },
  { code: 'KRW', symbol: '₩', decimals: 0, name: 'Korean Won' },
  { code: 'AUD', symbol: 'A$', decimals: 2, name: 'Australian Dollar' },
  { code: 'CAD', symbol: 'C$', decimals: 2, name: 'Canadian Dollar' },
  { code: 'CHF', symbol: 'CHF', decimals: 2, name: 'Swiss Franc' },
  { code: 'CNY', symbol: '¥', decimals: 2, name: 'Chinese Yuan' },
  { code: 'SEK', symbol: 'kr', decimals: 2, name: 'Swedish Krona' }
]
