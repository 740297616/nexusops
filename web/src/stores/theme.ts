import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const dark = ref(false)

  function apply() {
    document.documentElement.classList.toggle('dark', dark.value)
    document.documentElement.style.colorScheme = dark.value ? 'dark' : 'light'
  }

  function initialize() {
    try {
      const saved = localStorage.getItem('nexusops-theme')
      dark.value = saved
        ? saved === 'dark'
        : window.matchMedia('(prefers-color-scheme: dark)').matches
    } catch {
      dark.value = false
    }
    apply()
  }

  function toggle() {
    dark.value = !dark.value
    apply()
    try {
      localStorage.setItem('nexusops-theme', dark.value ? 'dark' : 'light')
    } catch {
      // The current session still uses the selected theme when storage is unavailable.
    }
  }

  return { dark, initialize, toggle }
})
