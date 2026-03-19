// Validation constants
export const MIN_NAME_LENGTH = 3
export const MIN_SLUG_LENGTH = 3
export const MIN_MERCHANT_ID_LENGTH = 36
export const MIN_PROJECT_ID_LENGTH = 36
export const MIN_PRIVATE_KEY_LENGTH = 1500
export const MIN_SECRET_KEY_LENGTH = 100
export const MIN_CLIENT_ID_LENGTH = 80
export const MIN_PAYPAL_SECRET_KEY_LENGTH = 80
export const MIN_COINBASE_API_KEY_LENGTH = 20

// Error messages
export const ERROR_MESSAGES = {
  NAME_TOO_SHORT: `Name must be at least ${MIN_NAME_LENGTH} characters`,
  SLUG_TOO_SHORT: `Slug must be at least ${MIN_SLUG_LENGTH} characters`,
  MERCHANT_ID_TOO_SHORT: `Merchant ID must be at least ${MIN_MERCHANT_ID_LENGTH} characters`,
  PROJECT_ID_TOO_SHORT: `Project ID must be at least ${MIN_PROJECT_ID_LENGTH} characters`,
  PRIVATE_KEY_TOO_SHORT: `Private key must be at least ${MIN_PRIVATE_KEY_LENGTH} characters`,
  SECRET_KEY_TOO_SHORT: `Secret key must be at least ${MIN_SECRET_KEY_LENGTH} characters`,
  CLIENT_ID_TOO_SHORT: `Client ID must be at least ${MIN_CLIENT_ID_LENGTH} characters`,
  PAYPAL_SECRET_KEY_TOO_SHORT: `Secret key must be at least ${MIN_PAYPAL_SECRET_KEY_LENGTH} characters`,
  COINBASE_API_KEY_TOO_SHORT: `API key must be at least ${MIN_COINBASE_API_KEY_LENGTH} characters`,
  AMOUNT_INVALID: 'Amount must be a non-negative number',
  DIGITAL_TYPE_REQUIRED: 'Digital type is required'
} as const
