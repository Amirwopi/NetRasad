import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import {
  DEFAULT_SETTINGS,
  type AppSettings,
  type ThemeMode,
  type UnitSystem,
} from '@/types'
import { setLocale, type AppLocale } from '@/i18n'

const STORAGE_KEY = 'netrasad.settings'

function loadFromStorage(): AppSettings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...DEFAULT_SETTINGS }
    return { ...DEFAULT_SETTINGS, ...(JSON.parse(raw) as Partial<AppSettings>) }
  } catch {
    return { ...DEFAULT_SETTINGS }
  }
}

export const useSettingsStore = defineStore('settings', () => {
  const settings = ref<AppSettings>(loadFromStorage())

  const theme = computed(() => settings.value.theme)
  const locale = computed(() => settings.value.locale)

  watch(theme, (t) => {
    if (typeof document !== 'undefined') {
      document.documentElement.setAttribute('data-theme', t)
    }
  }, { immediate: true })

  watch(settings, (s) => {
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(s)) } catch {}
  }, { deep: true })

  function load(): void {
    settings.value = loadFromStorage()
    setLocale(settings.value.locale)
  }

  function setTheme(t: ThemeMode): void {
    settings.value.theme = t
  }

  function toggleTheme(): void {
    settings.value.theme = settings.value.theme === 'dark' ? 'light' : 'dark'
  }

  function setLocaleValue(l: AppLocale): void {
    settings.value.locale = l
    setLocale(l)
  }

  function toggleLocale(): void {
    const next: AppLocale = settings.value.locale === 'en' ? 'fa' : 'en'
    setLocaleValue(next)
  }

  function setUnits(u: UnitSystem): void {
    settings.value.units = u
  }

  return {
    settings,
    theme,
    locale,
    load,
    setTheme,
    toggleTheme,
    setLocale: setLocaleValue,
    toggleLocale,
    setUnits,
  }
})
