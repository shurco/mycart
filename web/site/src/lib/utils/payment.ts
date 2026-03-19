/**
 * Utility for working with payment providers
 */

import type { PaymentMethods } from '$lib/types/models'

const PROVIDER_KEYS = ['stripe', 'paypal', 'spectrocoin', 'coinbase'] as const

export function getAvailableProviders(payments: PaymentMethods): string[] {
  return PROVIDER_KEYS.filter((key) => payments[key])
}

export function autoSelectProvider(payments: PaymentMethods): string {
  const providers = getAvailableProviders(payments)
  return providers.length === 1 ? providers[0] : ''
}

export function hasPaymentProviders(payments: PaymentMethods): boolean {
  return getAvailableProviders(payments).length > 0
}
