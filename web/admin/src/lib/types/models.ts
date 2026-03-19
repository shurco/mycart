export interface Product {
  id: string
  name: string
  slug: string
  brief?: string
  description?: string
  amount: number | string
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

export interface PaymentSettings {
  currency: string
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
