import { writable, derived, get } from 'svelte/store'
import zh from './locales/zh.json'
import en from './locales/en.json'
import ko from './locales/ko.json'

export type Locale = 'zh' | 'en' | 'ko'

const translations: Record<Locale, any> = {
  zh,
  en,
  ko
}

const defaultLocale: Locale = 'en'

// Store for current locale
function createLocaleStore() {
  const { subscribe, set, update } = writable<Locale>(defaultLocale)

  // Load locale from localStorage on initialization
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem('locale') as Locale
    if (saved && (saved === 'zh' || saved === 'en' || saved === 'ko')) {
      set(saved)
    }
  }

  return {
    subscribe,
    set: (locale: Locale) => {
      set(locale)
      if (typeof window !== 'undefined') {
        localStorage.setItem('locale', locale)
      }
    },
    update
  }
}

export const locale = createLocaleStore()

// Helper function to get translation
function getTranslation(localeValue: Locale, key: string, params?: Record<string, string | number>): string {
  const keys = key.split('.')
  let value: any = translations[localeValue]

  for (const k of keys) {
    if (value && typeof value === 'object' && k in value) {
      value = value[k]
    } else {
      // Fallback to English if key not found
      value = translations.en
      for (const fallbackKey of keys) {
        if (value && typeof value === 'object' && fallbackKey in value) {
          value = value[fallbackKey]
        } else {
          return key // Return key if translation not found
        }
      }
      break
    }
  }

  if (typeof value !== 'string') {
    return key
  }

  // Replace parameters in string
  if (params) {
    return value.replace(/\{\{(\w+)\}\}/g, (match, paramKey) => {
      return params[paramKey]?.toString() || match
    })
  }

  return value
}

// Function to get translation by key (non-reactive, for use outside components)
export function t(key: string, params?: Record<string, string | number>): string {
  const currentLocale = get(locale)
  return getTranslation(currentLocale, key, params)
}

// Derived store for reactive access to translations
export const translate = derived(locale, (currentLocale) => {
  return (key: string, params?: Record<string, string | number>) => {
    return getTranslation(currentLocale, key, params)
  }
})

// List of available locales with localized names
export function getAvailableLocales(currentLocale: Locale): Array<{ code: Locale; name: string }> {
  const localeNames: Record<Locale, Record<Locale, string>> = {
    en: {
      en: 'English',
      zh: '中文',
      ko: '한국어'
    },
    zh: {
      en: 'English',
      zh: '中文',
      ko: '한국어'
    },
    ko: {
      en: 'English',
      zh: '中文',
      ko: '한국어'
    }
  }
  
  return [
    { code: 'en', name: localeNames[currentLocale].en },
    { code: 'zh', name: localeNames[currentLocale].zh },
    { code: 'ko', name: localeNames[currentLocale].ko }
  ]
}

// Derived store for available locales with localized names
export const availableLocales = derived(locale, (currentLocale) => {
  return getAvailableLocales(currentLocale)
})
