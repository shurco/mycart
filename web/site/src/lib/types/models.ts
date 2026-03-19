export interface Product {
  id: string
  name: string
  slug: string
  amount: number
  brief?: string
  description?: string
  images?: Array<{ name: string; ext: string }>
  attributes?: string[]
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
  image?: { name: string; ext: string } | null
}

export interface Settings {
  main: {
    site_name: string
    domain: string
    currency: string
  }
  socials: Record<string, string>
  pages: Page[]
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
}
