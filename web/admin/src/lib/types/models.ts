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
  images?: Array<{
    id: string
    name: string
    ext: string
    orig_name: string
  }>
}

export interface Product {
  id: string
  name: string
  slug: string
  brief?: string
  description?: string
  amount: number | string
  quantity?: number
  sku?: string
  has_variants?: boolean
  active: boolean
  created?: string
  updated?: string
  metadata?: Array<{ key: string; value: string }>
  attributes?: string[]
  digital?: {
    type: 'file' | 'data' | 'api' | ''
    filled?: boolean
  }
  images?: Array<{
    id: string
    name: string
    ext: string
    orig_name: string
  }>
  options?: ProductOption[]
  variants?: ProductVariant[]
  seo?: {
    title?: string
    keywords?: string
    description?: string
  }
}

export interface Page {
  id: string
  name: string
  slug: string
  position: 'header' | 'footer'
  content?: string
  active: boolean
  created?: string | number
  updated?: string | number
  seo?: {
    title?: string
    keywords?: string
    description?: string
  }
}

export interface Cart {
  id: string
  email: string
  amount_total: number
  currency: string
  payment_status: 'paid' | 'pending' | 'failed'
  payment_system?: string
  payment_id?: string
  created?: string
  updated?: string
}

export interface CartItem {
  id: string
  name: string
  slug: string
  amount: number
  quantity: number
  image?: {
    id: string
    name: string
    ext: string
    orig_name?: string
  }
}

export interface CartDetail extends Cart {
  items?: CartItem[]
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

export interface StripeSettings {
  active: boolean
  secret_key: string
}

export interface PaypalSettings {
  active: boolean
  client_id: string
  secret_key: string
}

export interface SpectrocoinSettings {
  active: boolean
  merchant_id: string
  project_id: string
  private_key: string
}

export interface CoinbaseSettings {
  active: boolean
  api_key: string
}

export interface PortoneSettings {
  active: boolean
  store_id: string
  channel_key: string
  api_secret: string
  debug_enabled?: boolean
  supported_currencies?: string[]
}

export interface SmtpSettings {
  host: string
  port: string
  encryption: string
  username: string
  password: string
}

export interface LetterData {
  id: string
  key: string
  value: string
}

export interface LetterContent {
  subject: string
  text: string
  html: string
}
