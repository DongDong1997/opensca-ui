import {defineStore} from 'pinia'
import {ref} from 'vue'

const THEME_KEY = 'opensca-ui:theme'
const SIDEBAR_KEY = 'opensca-ui:sidebar'
const LOG_AUTOSCROLL_KEY = 'opensca-ui:log-autoscroll'

function loadLocal<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(key)
    if (raw == null) return fallback
    return JSON.parse(raw) as T
  } catch {
    return fallback
  }
}

function saveLocal(key: string, value: unknown) {
  try {
    localStorage.setItem(key, JSON.stringify(value))
  } catch {
    /* ignore */
  }
}

export const useUIStore = defineStore('ui', () => {
  const theme = ref<'light' | 'dark'>(loadLocal<'light' | 'dark'>(THEME_KEY, 'light'))
  const sidebarCollapsed = ref<boolean>(loadLocal<boolean>(SIDEBAR_KEY, false))
  const logAutoScroll = ref<boolean>(loadLocal<boolean>(LOG_AUTOSCROLL_KEY, true))

  function setTheme(t: 'light' | 'dark') {
    theme.value = t
    saveLocal(THEME_KEY, t)
  }

  function toggleTheme() {
    setTheme(theme.value === 'dark' ? 'light' : 'dark')
  }

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
    saveLocal(SIDEBAR_KEY, sidebarCollapsed.value)
  }

  function setLogAutoScroll(v: boolean) {
    logAutoScroll.value = v
    saveLocal(LOG_AUTOSCROLL_KEY, v)
  }

  return {theme, sidebarCollapsed, logAutoScroll, setTheme, toggleTheme, toggleSidebar, setLogAutoScroll}
})