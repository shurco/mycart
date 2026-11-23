import type { Product } from '$lib/types/models'

export function digitalTypeIco(type: string | undefined): string {
  if (!type) return 'cube-transparent'
  switch (type) {
    case 'file':
      return 'paper-clip'
    case 'data':
      return 'queue-list'
    default:
      return 'cube-transparent'
  }
}

// Preserve digital.filled from original product if it wasn't in response
export function preserveDigitalFilled(product: Product, original: Product) {
  const filled = original.digital?.filled
  if (filled !== undefined && (!product.digital || product.digital.filled === undefined)) {
    product.digital = product.digital
      ? { ...product.digital, filled }
      : { type: original.digital?.type || '', filled }
  }
}
