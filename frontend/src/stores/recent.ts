import {defineStore} from 'pinia'
import {ref} from 'vue'
import {api} from '@/api'
import type {RecentProject} from '@/api/types'

export const useRecentStore = defineStore('recent', () => {
  const list = ref<RecentProject[]>([])
  const loading = ref(false)

  async function refresh() {
    loading.value = true
    try {
      list.value = await api.GetRecentProjects()
    } catch (e) {
      console.warn('GetRecentProjects failed:', e)
      list.value = []
    } finally {
      loading.value = false
    }
  }

  async function remove(path: string) {
    try {
      await api.RemoveRecentProject(path)
      list.value = list.value.filter((e) => e.path !== path)
    } catch (e) {
      console.warn('RemoveRecentProject failed:', e)
    }
  }

  async function clear() {
    try {
      await api.ClearRecentProjects()
      list.value = []
    } catch (e) {
      console.warn('ClearRecentProjects failed:', e)
    }
  }

  // 提交扫描时后端会自动记录；这里留一个手动入口，方便 HomeView "加入最近" 按钮。
  async function add(path: string, label?: string) {
    if (!path) return
    try {
      await api.AddRecentProject(path, label ?? '')
      await refresh()
    } catch (e) {
      console.warn('AddRecentProject failed:', e)
    }
  }

  return {
    list,
    loading,
    refresh,
    remove,
    clear,
    add
  }
})
