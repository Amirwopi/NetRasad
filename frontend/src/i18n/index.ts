import { createI18n } from 'vue-i18n'
import en from './en'
import fa from './fa'

export type AppLocale = 'en' | 'fa'

export const SUPPORTED_LOCALES: { value: AppLocale; label: string; dir: 'ltr' | 'rtl' }[] = [
  { value: 'en', label: 'English', dir: 'ltr' },
  { value: 'fa', label: 'فارسی', dir: 'rtl' },
]

const STORAGE_KEY = 'netrasad.locale'

function detectLocale(): AppLocale {
  try {
    const saved = localStorage.getItem(STORAGE_KEY) as AppLocale | null
    if (saved && (saved === 'en' || saved === 'fa')) return saved
  } catch {
  }
  const nav = typeof navigator !== 'undefined' ? navigator.language : 'en'
  return nav && nav.toLowerCase().startsWith('fa') ? 'fa' : 'en'
}

const initial = detectLocale()

export const i18n = createI18n({
  legacy: false,
  locale: initial,
  fallbackLocale: 'en',
  messages: { en, fa },
})

export function applyLocale(locale: AppLocale): void {
  const entry = SUPPORTED_LOCALES.find((l) => l.value === locale) ?? SUPPORTED_LOCALES[0]
  if (typeof document !== 'undefined') {
    document.documentElement.setAttribute('lang', locale)
    document.documentElement.setAttribute('dir', entry.dir)
  }
  try {
    localStorage.setItem(STORAGE_KEY, locale)
  } catch {
  }
}

export function setLocale(locale: AppLocale): void {
  i18n.global.locale.value = locale
  applyLocale(locale)
}

applyLocale(initial)
