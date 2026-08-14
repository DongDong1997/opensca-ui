import {computed} from 'vue'
import {useUIStore} from '@/stores/ui'

export function useTheme() {
  const ui = useUIStore()
  return {
    theme: computed(() => ui.theme),
    isDark: computed(() => ui.theme === 'dark'),
    setTheme: (t: 'light' | 'dark') => ui.setTheme(t),
    toggle: () => ui.toggleTheme()
  }
}