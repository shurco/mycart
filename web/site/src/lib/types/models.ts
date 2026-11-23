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

/**
 * How a product is delivered.
 *
 * The public payload carries the type and nothing else: the names of the files
 * and the licence keys behind them are not the storefront's to know.
 */
export interface Digital {
  type: string
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
  images?: Array<{ id?: string; name: string; ext: string }>
  attributes?: string[]
  options?: ProductOption[]
  variants?: ProductVariant[]
  digital?: Digital
  seo?: {
    title?: string
    keywords?: string
    description?: string
  }
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
  // Set by applyCartCorrections when the server reports the line is no longer sold
  needsDeletion?: boolean
  // What the line is, carried from the product so a cart that outlives the tab
  // still knows whether it holds a download. Items put in a cart before this
  // was stored have none, and are read as goods the shop ships — which is what
  // they were shown as when they were added.
  digital?: Digital
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
  // The cabinet switch. Optional because a settings payload cached before the
  // feature existed carries no such key; the storefront treats a missing one as
  // off, so an old payload cannot grow a link to a cabinet that is not there.
  account?: AccountSettings
  // The shop's own marks. Optional for the same reason as the cabinet switch:
  // a payload cached before this existed carries no such key, and the
  // storefront falls back to the mark the build shipped with.
  branding?: BrandingSettings
}

export interface AccountSettings {
  enabled: boolean
}

export interface BrandingSettings {
  /** Address of the uploaded logo, or an empty string when there is none. */
  logo: string
  /** Address of the uploaded favicon, or an empty string when there is none. */
  favicon: string
  tagline: string
}

export interface Customer {
  id: string
  email: string
  name?: string
}

export interface CustomerPurchaseFile {
  id: string
  orig_name: string
}

export interface CustomerPurchaseItem {
  product_id: string
  name: string
  slug: string
  quantity: number
  // "file", "data", "api" or absent. Files come with the first, codes with the
  // second, and an api product is delivered outside the cabinet.
  digital?: string
  files?: CustomerPurchaseFile[]
  codes?: string[]
}

export interface CustomerPurchase {
  id: string
  created: number
  amount_total: number
  currency: string
  items?: CustomerPurchaseItem[]
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
