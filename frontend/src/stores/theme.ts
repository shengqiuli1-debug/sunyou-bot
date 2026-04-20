import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  DEFAULT_THEME_ID,
  THEME_OPTIONS,
  THEME_STORAGE_KEY,
  getThemeOption,
  resolveThemeId,
  type ThemeId
} from '@/constants/themes'

export const useThemeStore = defineStore('theme', () => {
  const themeId = ref<ThemeId>(DEFAULT_THEME_ID)

  const currentTheme = computed(() => getThemeOption(themeId.value))

  function applyTheme(id: ThemeId) {
    if (typeof document === 'undefined') return
    const theme = getThemeOption(id)
    document.documentElement.setAttribute('data-theme', id)
    document.documentElement.style.colorScheme = theme.mode
  }

  function setTheme(id: ThemeId, persist = true) {
    const next = resolveThemeId(id)
    themeId.value = next
    applyTheme(next)
    if (persist && typeof window !== 'undefined') {
      window.localStorage.setItem(THEME_STORAGE_KEY, next)
    }
  }

  function initTheme() {
    if (typeof window === 'undefined') {
      setTheme(DEFAULT_THEME_ID, false)
      return
    }
    const saved = window.localStorage.getItem(THEME_STORAGE_KEY)
    setTheme(resolveThemeId(saved), false)
  }

  return {
    themeId,
    currentTheme,
    options: THEME_OPTIONS,
    initTheme,
    setTheme
  }
})
