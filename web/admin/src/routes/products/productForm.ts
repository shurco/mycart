import { validators, validateFields } from '$lib/utils/validation'
import { MIN_NAME_LENGTH, MIN_SLUG_LENGTH, ERROR_MESSAGES } from '$lib/constants/validation'
import { CENTS_PER_UNIT, DEFAULT_AMOUNT } from '$lib/constants/pricing'
import type { Product, ProductOption, ProductVariant } from '$lib/types/models'

export interface ProductFormData {
  name: string
  slug: string
  brief: string
  description: string
  amount: string | number
  quantity?: number
  sku?: string
  has_variants?: boolean
  active: boolean
  metadata: Array<{ key: string; value: string }>
  attributes: string[]
  digital: {
    type: '' | 'file' | 'data' | 'api'
  }
  options?: ProductOption[]
  variants?: ProductVariant[]
}

export function emptyProductForm(): ProductFormData {
  return {
    name: '',
    slug: '',
    brief: '',
    description: '',
    amount: DEFAULT_AMOUNT,
    quantity: 0,
    sku: '',
    has_variants: false,
    active: true,
    metadata: [],
    attributes: [],
    digital: {
      type: ''
    },
    options: [],
    variants: []
  }
}

// Converts Product from API to form data format with price conversion
export function convertProductToFormData(product: Product): ProductFormData {
  const amountValue = typeof product.amount === 'string' ? parseFloat(product.amount) : (product.amount || 0)
  const amountInUnits = amountValue / CENTS_PER_UNIT
  const amountStr = amountInUnits.toFixed(2)

  return {
    name: product.name || '',
    slug: product.slug || '',
    brief: product.brief || '',
    description: product.description || '',
    amount: amountStr,
    quantity: product.quantity || 0,
    sku: product.sku || '',
    has_variants: product.has_variants || false,
    active: product.active !== undefined ? product.active : true,
    metadata: product.metadata || [],
    attributes: product.attributes || [],
    digital: product.digital || { type: '' },
    options: product.options || [],
    variants: product.variants || []
  }
}

// Validates product form fields and returns error object
export function validateProductForm(formData: ProductFormData, drawerMode: string): Record<string, string> {
  const errors = validateFields(formData, [
    { field: 'name', ...validators.minLength(MIN_NAME_LENGTH, ERROR_MESSAGES.NAME_TOO_SHORT) },
    { field: 'slug', ...validators.minLength(MIN_SLUG_LENGTH, ERROR_MESSAGES.SLUG_TOO_SHORT) }
  ])

  const amountValue = typeof formData.amount === 'string' ? parseFloat(formData.amount) : formData.amount
  if (isNaN(amountValue) || amountValue < 0) {
    errors.amount = ERROR_MESSAGES.AMOUNT_INVALID
  }

  if (drawerMode === 'add' && (!formData.digital?.type || formData.digital.type.trim() === '')) {
    errors.digital_type = ERROR_MESSAGES.DIGITAL_TYPE_REQUIRED
  }

  return errors
}

// Prepares form data for API submission with conversions
export function prepareSubmitData(formData: ProductFormData): Partial<Product> {
  const amountValue = typeof formData.amount === 'string' ? parseFloat(formData.amount) : formData.amount
  const amountInCents = Math.round((amountValue || 0) * CENTS_PER_UNIT)

  // Convert Svelte 5 $state proxies to plain objects for JSON serialization
  return {
    ...formData,
    amount: amountInCents,
    options: formData.options ? JSON.parse(JSON.stringify(formData.options)) : [],
    variants: formData.variants ? JSON.parse(JSON.stringify(formData.variants)) : []
  }
}

// Keeps digits and a single dot, with a leading zero before a bare dot
export function sanitizeAmountInput(raw: string): string {
  // Remove invalid characters (keep only digits and dot)
  let value = raw.replace(/[^0-9.]/g, '')

  // Add leading zero if starts with dot
  if (value.startsWith('.')) {
    value = '0' + value
  }

  // Limit to single dot
  const parts = value.split('.')
  if (parts.length > 2) {
    value = parts[0] + '.' + parts.slice(1).join('')
  }

  return value
}
