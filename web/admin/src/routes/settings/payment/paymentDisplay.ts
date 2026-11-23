import { CURRENCIES } from '$lib/config/currencies'
import type { TruncationSettings } from '$lib/types/models'

// Settings version key for cache invalidation (localStorage is shared across tabs)
const SETTINGS_VERSION_KEY = 'settings_version'

// Increment settings version to invalidate storefront cache
export function incrementSettingsVersion() {
  if (typeof localStorage === 'undefined') return
  const currentVersion = parseInt(localStorage.getItem(SETTINGS_VERSION_KEY) || '1', 10)
  localStorage.setItem(SETTINGS_VERSION_KEY, (currentVersion + 1).toString())
}

// Initialize truncation settings with defaults
export const defaultTruncation = (): TruncationSettings => ({
  admin: {},
  storefront: {}
})

// Ensure all currencies have default 'none' mode
export function ensureDefaults(truncation: TruncationSettings | undefined): TruncationSettings {
  const result = truncation || defaultTruncation()

  CURRENCIES.forEach(curr => {
    if (!result.admin[curr.code]) {
      result.admin[curr.code] = { mode: 'none' }
    }
    if (!result.storefront[curr.code]) {
      result.storefront[curr.code] = { mode: 'none' }
    }
  })

  return result
}
