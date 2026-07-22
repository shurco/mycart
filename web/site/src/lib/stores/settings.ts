import { writable } from 'svelte/store'
import type { Settings } from '$lib/types/models'
import { isBrowser } from '$lib/utils/browser'

const CACHE_DURATION = 5 * 60 * 1000 // 5 minutes
const SETTINGS_VERSION_KEY = 'settings_version'

function createSettingsStore() {
  const { subscribe, set, update } = writable<Settings | null>(null)

  return {
    subscribe,
    set,
    update,
    loadFromCache: (): Settings | null => {
      if (!isBrowser()) return null

      const cached = sessionStorage.getItem('settings')
      const timestamp = sessionStorage.getItem('settings_timestamp')
      const cachedVersion = sessionStorage.getItem('settings_cached_version')
      const currentVersion = sessionStorage.getItem(SETTINGS_VERSION_KEY)

      // If version has changed, invalidate cache
      if (cachedVersion && currentVersion && cachedVersion !== currentVersion) {
        return null
      }

      if (cached && timestamp) {
        const now = Date.now()
        const cacheTime = parseInt(timestamp, 10)

        if (now < cacheTime) {
          try {
            return JSON.parse(cached)
          } catch {
            return null
          }
        }
      }

      return null
    },
    saveToCache: (settings: Settings) => {
      if (!isBrowser()) return

      const expiry = Date.now() + CACHE_DURATION
      const currentVersion = sessionStorage.getItem(SETTINGS_VERSION_KEY) || '1'

      sessionStorage.setItem('settings', JSON.stringify(settings))
      sessionStorage.setItem('settings_timestamp', expiry.toString())
      sessionStorage.setItem('settings_cached_version', currentVersion)
    },
    clearCache: () => {
      if (!isBrowser()) return

      sessionStorage.removeItem('settings')
      sessionStorage.removeItem('settings_timestamp')
      sessionStorage.removeItem('settings_cached_version')
    },
    incrementVersion: () => {
      if (!isBrowser()) return

      const currentVersion = parseInt(sessionStorage.getItem(SETTINGS_VERSION_KEY) || '1', 10)
      sessionStorage.setItem(SETTINGS_VERSION_KEY, (currentVersion + 1).toString())
    }
  }
}

export const settingsStore = createSettingsStore()
