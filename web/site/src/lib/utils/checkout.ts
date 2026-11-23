import * as PortOne from '@portone/browser-sdk/v2'
import { goto } from '$app/navigation'
import { cartStore } from '$lib/stores/cart'
import { apiGet, apiPost } from '$lib/utils/api'
import { removeLocalStorage } from '$lib/utils/browser'
import type { PaymentMethods } from '$lib/types/models'

type Log = (...args: any[]) => void
type ValidationHandler = (errors: any[], correctedCart: any[]) => void

export interface PortoneConfig {
  storeId: string
  channelKey: string
}

export async function fetchPaymentMethods(): Promise<{
  payments: PaymentMethods
  portone?: PortoneConfig & { debugEnabled: boolean }
}> {
  const res = await apiGet<PaymentMethods>('/api/cart/payment')
  if (!res.success || !res.result) {
    throw new Error(res.message || 'Failed to load payment methods')
  }
  const payments = res.result

  // Load PortOne config if portone is available
  if (payments.portone) {
    const portoneRes = await apiGet<{ store_id: string; channel_key: string; debug_enabled: boolean }>('/api/cart/portone-config')
    if (portoneRes.success && portoneRes.result) {
      return {
        payments,
        portone: {
          storeId: portoneRes.result.store_id,
          channelKey: portoneRes.result.channel_key,
          debugEnabled: portoneRes.result.debug_enabled
        }
      }
    }
  }
  return { payments }
}

export function postStandardCheckout(email: string, provider: string, cart: any[]) {
  return apiPost<{ url?: string; validation_errors?: any[]; corrected_cart?: any[] }>('/cart/payment', {
    email,
    provider,
    products: cart.map((item) => ({
      id: item.id,
      variant_id: item.variant_id || undefined,
      quantity: item.quantity
    }))
  })
}

export function getItemKey(item: any): string {
  return item.variant_id ? `${item.id}_${item.variant_id}` : item.id
}

// Apply the server's corrected prices/availability to the cart; also report
// which lines to highlight.
export function applyCartCorrections(cart: any[], errors: any[], correctedCart: any[]) {
  const highlighted: string[] = []
  const updated = cart.map((item, index) => {
    const corrected = correctedCart[index]
    if (!corrected) return item

    const hasError = errors.some(e => e.item_index === index)

    if (!corrected.available) {
      return { ...item, needsDeletion: true, disabled: true }
    }

    if (hasError) {
      highlighted.push(getItemKey(item))
      return { ...item, amount: corrected.unit_price }
    }

    return item
  })
  return { cart: updated, highlighted }
}

// UUID generator with fallback for non-secure contexts (HTTP)
export function generateUUID(): string {
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  // Fallback for HTTP or older browsers
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = Math.random() * 16 | 0, v = c === "x" ? r : (r & 0x3 | 0x8)
    return v.toString(16)
  })
}

// Create cart record and return cart_id
async function createCartRecord(
  email: string,
  cart: any[],
  onValidationErrors: ValidationHandler,
  debugLog: Log
): Promise<string> {
  debugLog('Creating cart record...')
  const cartCreateRes = await apiPost<{
    cart_id: string
    amount_total: number
    currency: string
    validation_errors?: any[]
    corrected_cart?: any[]
  }>('/api/cart/create', {
    email: email,
    provider: 'portone',
    products: cart.map((item) => ({
      id: item.id,
      variant_id: item.variant_id || undefined,
      quantity: item.quantity,
      unit_price: item.amount
    }))
  })
  debugLog('Cart create response:', cartCreateRes)

  // Handle validation errors (409 Conflict)
  if (cartCreateRes.status === 409) {
    if (!cartCreateRes.result?.validation_errors || !cartCreateRes.result?.corrected_cart) {
      throw new Error('Validation error occurred. Please refresh and try again.')
    }
    onValidationErrors(
      cartCreateRes.result.validation_errors,
      cartCreateRes.result.corrected_cart
    )
    throw new Error('Cart validation failed')
  }

  if (!cartCreateRes.success || !cartCreateRes.result?.cart_id) {
    throw new Error('Failed to create cart: ' + (cartCreateRes.message || 'Unknown error'))
  }

  return cartCreateRes.result.cart_id
}

// Verify payment with backend
async function verifyPayment(paymentId: string, cartId: string): Promise<boolean> {
  const verifyRes = await apiPost('/api/payment/portone/complete', {
    payment_id: paymentId,
    cart_id: cartId
  })

  if (!verifyRes.success) {
    throw new Error('Payment verification failed: ' + (verifyRes.message || 'Unknown error'))
  }

  return true
}

// Handle PortOne payment flow
export async function handlePortonePayment(
  config: PortoneConfig,
  email: string,
  cart: any[],
  cartTotal: number,
  currency: string,
  onValidationErrors: ValidationHandler,
  debugLog: Log
) {
  debugLog('=== PORTONE PAYMENT FLOW ===')
  debugLog('PortOne SDK available?', typeof PortOne !== 'undefined')
  debugLog('PortOne.requestPayment available?', typeof PortOne?.requestPayment === 'function')

  // Validate PortOne configuration
  if (!config.storeId || !config.channelKey) {
    throw new Error('PortOne configuration not loaded. Please refresh the page.')
  }

  debugLog('PortOne payment request:', {
    storeId: config.storeId,
    channelKey: config.channelKey,
    cartTotal: cartTotal,
    currency: currency,
    protocol: window.location.protocol,
    hostname: window.location.hostname
  })

  // Generate unique payment ID
  const paymentId = `payment-${generateUUID()}`
  debugLog('Generated payment ID:', paymentId)

  // Create cart and get cart_id
  const cartId = await createCartRecord(email, cart, onValidationErrors, debugLog)
  debugLog('Cart created with ID:', cartId)

  // Prepare and execute payment request.
  // Currency comes from store settings (backend gates PortOne by
  // SupportedCurrencies); payMethod is only meaningful for KRW channels —
  // otherwise the PortOne payment window lets the buyer choose.
  const paymentRequest: Partial<PortOne.PaymentRequest> = {
    storeId: config.storeId,
    channelKey: config.channelKey,
    paymentId: paymentId,
    orderName: `Order ${cart.length} items`,
    totalAmount: cartTotal,
    currency: currency.toUpperCase() as PortOne.Currency,
    customData: { cart_id: cartId }
  }
  if (currency.toUpperCase() === 'KRW') {
    paymentRequest.payMethod = 'EASY_PAY'
  }
  debugLog('Payment request object:', paymentRequest)

  // Call PortOne SDK
  // payMethod is set only for KRW (above), so the Partial is narrowed back to the SDK's type here.
  const response = await PortOne.requestPayment(paymentRequest as PortOne.PaymentRequest)
  debugLog('PortOne payment response received:', response)

  // The SDK resolves with nothing when the payment window is closed without a result
  if (!response) {
    throw new Error('Payment was not completed')
  }

  // Check for payment errors
  if (response.code != null) {
    throw new Error(response.message)
  }

  // Verify payment with backend
  await verifyPayment(response.paymentId, cartId)

  // Clear cart and redirect to success
  cartStore.set([])
  removeLocalStorage('email')
  removeLocalStorage('provider')
  goto('/cart/payment/success')
}
