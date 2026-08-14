<script setup lang="ts">
import {computed} from 'vue'
import {useRouter} from 'vue-router'
import {useI18n} from 'vue-i18n'
import {NEmpty, NSpace, NIcon, NText, NTag, NPagination, useMessage} from 'naive-ui'
import {FolderOutline} from '@vicons/ionicons5'
import AppShell from '@/components/AppShell.vue'
import TaskCard from '@/components/TaskCard.vue'
import {useTasksStore} from '@/stores/tasks'
import {usePagination} from '@/composables/usePagination'
import type {Task} from '@/api/types'

const tasks = useTasksStore()
const router = useRouter()
const message = useMessage()
const {t} = useI18n()

// 每个文件夹的最新任务：用来识别哪些任务是"历史"
const latestByFolder = computed<Map<string, Task>>(() => {
  const map = new Map<string, Task>()
  for (const t of tasks.list) {
    const cur = map.get(t.path)
    if (!cur || t.startedAt > cur.startedAt) {
      map.set(t.path, t)
    }
  }
  return map
})

// 历史任务（所有文件夹，排除每个文件夹的最新一条）
const allHistory = computed<Task[]>(() => {
  const latestIds = new Set(Array.from(latestByFolder.value.values()).map((t) => t.id))
  return tasks.list.filter((t) => !latestIds.has(t.id))
})

// 每个文件夹的历史数量（用于顶部 "历史 X 条" 角标）
const historyCountByPath = computed<Map<string, number>>(() => {
  const counts = new Map<string, number>()
  for (const t of allHistory.value) {
    counts.set(t.path, (counts.get(t.path) ?? 0) + 1)
  }
  return counts
})

// 每个文件夹的历史中最新的一条（mirror TasksListView 的"按项目"——每组只挂最新一条）
const latestHistoryByFolder = computed<Task[]>(() => {
  const map = new Map<string, Task>()
  for (const t of allHistory.value) {
    const cur = map.get(t.path)
    if (!cur || t.startedAt > cur.startedAt) {
      map.set(t.path, t)
    }
  }
  return Array.from(map.values()).sort((a, b) => b.startedAt - a.startedAt)
})

// 分页：按"项目/文件夹组"切（一组 ≈ 一条历史缩略记录）
const {page, pageSize, pageSizes, pagedItems, total} = usePagination(latestHistoryByFolder)

function folderDisplayName(t: {path: string; projectName?: string; label?: string}): string {
  // 优先用稳定的 projectName（从路径推导的目录 basename 或压缩包去扩展名）。
  // 没有再退到 label，最后回退到 path basename。
  if (t.projectName && t.projectName.trim()) return t.projectName
  if (t.label && t.label.trim()) return t.label
  const parts = t.path.split(/[\\/]/).filter(Boolean)
  return parts[parts.length - 1] || t.path
}

// 点击"共 N 条" → 跳到挂接在 /history 下面的 per-folder 历史子页面
function goFolderHistory(path: string) {
  router.push({
    name: 'history-folder',
    params: {path: encodeURIComponent(path)}
  })
}

function viewTask(id: string) {
  // from=history-all 让 ReportView 的返回按钮跳回 /history
  router.push({name: 'report', params: {id}, query: {from: 'history-all'}})
}

async function cancelTask(id: string) {
  await tasks.cancel(id)
  message.info(t('tasksRunning.cancelSent'))
}

async function removeTask(id: string) {
  await tasks.remove(id)
}
</script>

<template>
  <AppShell>
    <NEmpty v-if="latestHistoryByFolder.length === 0" :description="t('history.noHistory')" style="margin-top: 60px" />
    <template v-else>
      <div class="group-list">
        <div v-for="item in pagedItems" :key="item.path" class="group-block">
          <div class="group-header">
            <NSpace align="center" :size="8">
              <NIcon :component="FolderOutline" :size="18" color="#2080f0" />
              <NText strong>{{ folderDisplayName(item) }}</NText>
              <NText depth="3" class="group-path">{{ item.path }}</NText>
              <NTag
                v-if="(historyCountByPath.get(item.path) ?? 0) > 0"
                size="tiny"
                round
                class="history-tag"
                @click="goFolderHistory(item.path)"
              >
                {{ t('history.entryCount', {n: historyCountByPath.get(item.path)}) }}
              </NTag>
            </NSpace>
          </div>
          <TaskCard
            :task="item"
            hide-progress
            @view="viewTask"
            @report="viewTask"
            @cancel-task="cancelTask"
            @removed="removeTask"
          />
        </div>
      </div>
      <!-- 列表分页器：组数 > 页大小时才显示 -->
      <div v-if="total > pageSize" class="list-pager">
        <NPagination
          v-model:page="page"
          :page-size="pageSize"
          :item-count="total"
          :page-sizes="pageSizes"
          show-size-picker
          size="small"
        />
      </div>
    </template>
  </AppShell>
</template>

<style scoped>
.group-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.group-block {
  background: var(--n-card-color);
  border: 1px solid var(--n-border-color);
  border-radius: 8px;
  padding: 12px 16px 4px;
}
.group-header {
  padding-bottom: 8px;
  margin-bottom: 4px;
  border-bottom: 1px dashed var(--n-border-color);
}
.group-path {
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 480px;
}
.history-tag {
  cursor: pointer;
  transition: opacity 0.15s ease;
}
.history-tag:hover {
  opacity: 0.75;
}
.list-pager {
  display: flex;
  justify-content: center;
  padding: 16px 0 8px;
}
</style>
