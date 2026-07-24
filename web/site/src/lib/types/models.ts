export interface ProductOptionValue {
  id?: string
  option_id?: string
  value: string
  position: number
}

export interface ProductOption {
  id?: string
  product_id?: string
  name: string
  values: ProductOptionValue[]
  position: number
}

export interface ProductVariant {
  id?: string
  product_id?: string
  sku?: string
  price_surcharge: number
  quantity: number
  option_values: Record<string, string>
  active: boolean
}

export interface Product {
  id: string
  name: string
  slug: string
  amount: number
  brief?: string
  description?: string
  has_variants?: boolean
  quantity?: number
  images?: Array<{ name: string; ext: string }>
  attributes?: string[]
  options?: ProductOption[]
  variants?: ProductVariant[]
  seo?: {
    title?: string
    keywords?: string
    description?: string
  }
  inCart?: boolean
}

export interface CartItem {
  id: string
  name: string
  slug: string
  amount: number
  quantity: number       // Number of this item in cart (min: 1)
  image?: { name: string; ext: string } | null
  variant_id?: string
  variant_name?: string
}

export interface Settings {
  main: {
    site_name: string
    domain: string
    currency: string
  }
  socials: Record<string, string>
  pages: Page[]
  payment?: PaymentSettings
}

export interface Page {
  id: string
  name: string
  slug: string
  position: string
  content: string
  seo?: {
    title?: string
    keywords?: string
    description?: string
  }
}

export interface PaymentMethods {
  stripe?: boolean
  paypal?: boolean
  spectrocoin?: boolean
  coinbase?: boolean
  portone?: boolean
}

export interface CurrencyTruncationSettings {
  mode: 'none' | 'fixed' | 'flexible'
  fixed_unit?: string  // e.g., 'K', 'M', '만', '천'
}

export interface NumberFormatSettings {
  decimal_precision: 0 | 1 | 2
  show_trailing_zeros: boolean
}

export interface SymbolDisplaySettings {
  admin: 'currency' | 'language'
  storefront: 'currency' | 'language'
}

export interface TruncationSettings {
  admin: Record<string, CurrencyTruncationSettings>
  storefront: Record<string, CurrencyTruncationSettings>
}

export interface PaymentSettings {
  currency: string
  truncation?: TruncationSettings
  number_format?: NumberFormatSettings
  symbol_display?: SymbolDisplaySettings
}
