<script setup lang="ts">
import {computed, onMounted} from 'vue'
import {useRouter, useRoute, RouterView} from 'vue-router'
import {NTabs, NTabPane} from 'naive-ui'
import AppShell from '@/components/AppShell.vue'
import {useTasksStore} from '@/stores/tasks'

const tasks = useTasksStore()
const router = useRouter()
const route = useRoute()

// 只剩两个 tab：
//   tasks-running  → 运行中（pending + running）
//   tasks-finished → 已完成（success + failed + canceled）
//
// 任务管理只承担"当前会话内"的任务视图；
// 持久化的全部扫描记录（按项目分组）由"历史记录"页面承担。
type TabKey = 'tasks-running' | 'tasks-finished'
const tabValue = computed<TabKey>({
  get() {
    return route.name === 'tasks-finished' ? 'tasks-finished' : 'tasks-running'
  },
  set(v) {
    router.push({name: v})
  }
})

// 各种状态过滤（用于 tab 角标）
// 只数本次会话内的任务，避免重启后还看到历史已完成的
const runningCount = computed(() =>
  tasks.currentSessionList.filter((t) => t.status === 'running' || t.status === 'pending').length
)
const finishedCount = computed(() =>
  tasks.currentSessionList.filter((t) => t.status === 'success' || t.status === 'failed' || t.status === 'canceled').length
)

onMounted(async () => {
  await tasks.refresh()
})
</script>

<template>
  <AppShell>
    <NTabs
      :value="tabValue"
      type="line"
      animated
      @update:value="(v: TabKey) => (tabValue = v)"
    >
      <NTabPane name="tasks-running" :tab="`运行中 (${runningCount})`" />
      <NTabPane name="tasks-finished" :tab="`已完成 (${finishedCount})`" />
    </NTabs>

    <!-- 子页面渲染区（运行中 / 已完成） -->
    <RouterView />
  </AppShell>
</template>

<style scoped>
/* 让 tab bar 跟下面的内容保持一点间距 */
:deep(.n-tabs) {
  margin-bottom: 16px;
}
</style>