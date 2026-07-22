import { writable } from 'svelte/store'
import type { PaymentSettings } from '$lib/types/models'

export const paymentSettingsStore = writable<PaymentSettings | null>(null)
