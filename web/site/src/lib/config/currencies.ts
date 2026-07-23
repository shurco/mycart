// web/site/src/lib/config/currencies.ts
export interface CurrencyConfig {
  code: string           // ISO 4217 code
  symbol: string         // Display symbol
  decimals: number       // Decimal places for display
  name: string          // English name
  names: {              // Localized names
    en: string
    ko: string
    zh: string
  }
}

export const CURRENCIES: CurrencyConfig[] = [
  {
    code: 'USD', symbol: '$', decimals: 2, name: 'US Dollar',
    names: { en: 'Dollar', ko: '달러', zh: '美元' }
  },
  {
    code: 'EUR', symbol: '€', decimals: 2, name: 'Euro',
    names: { en: 'Euro', ko: '유로', zh: '欧元' }
  },
  {
    code: 'GBP', symbol: '£', decimals: 2, name: 'British Pound',
    names: { en: 'Pound', ko: '파운드', zh: '英镑' }
  },
  {
    code: 'JPY', symbol: '¥', decimals: 0, name: 'Japanese Yen',
    names: { en: 'Yen', ko: '엔', zh: '日元' }
  },
  {
    code: 'KRW', symbol: '₩', decimals: 0, name: 'Korean Won',
    names: { en: 'Won', ko: '원', zh: '韩元' }
  },
  {
    code: 'AUD', symbol: 'A$', decimals: 2, name: 'Australian Dollar',
    names: { en: 'Dollar', ko: '달러', zh: '澳元' }
  },
  {
    code: 'CAD', symbol: 'C$', decimals: 2, name: 'Canadian Dollar',
    names: { en: 'Dollar', ko: '달러', zh: '加元' }
  },
  {
    code: 'CHF', symbol: 'CHF', decimals: 2, name: 'Swiss Franc',
    names: { en: 'Franc', ko: '프랑', zh: '瑞郎' }
  },
  {
    code: 'CNY', symbol: '¥', decimals: 2, name: 'Chinese Yuan',
    names: { en: 'Yuan', ko: '위안', zh: '元' }
  },
  {
    code: 'SEK', symbol: 'kr', decimals: 2, name: 'Swedish Krona',
    names: { en: 'Krona', ko: '크로나', zh: '克朗' }
  }
]
