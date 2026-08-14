<script setup lang="ts">
import {computed} from 'vue'
import {useRouter} from 'vue-router'
import {NButton, NEmpty, NIcon, NPagination, NSpace, NText, NTooltip, useMessage} from 'naive-ui'
import {ArrowBackOutline, FolderOpenOutline} from '@vicons/ionicons5'
import AppShell from '@/components/AppShell.vue'
import TaskCard from '@/components/TaskCard.vue'
import {useTasksStore} from '@/stores/tasks'
import {usePagination} from '@/composables/usePagination'
import type {Task} from '@/api/types'

const props = defineProps<{path: string}>()
const tasks = useTasksStore()
const router = useRouter()
const message = useMessage()

// 路径解码（路由过来的是 encodeURIComponent 过的）
const folderPath = computed(() => {
  try {
    return decodeURIComponent(props.path)
  } catch {
    return props.path
  }
})

// 该文件夹下的全部任务（按 startedAt 倒序）
const folderTasks = computed<Task[]>(() =>
  tasks.list
    .filter((t) => t.path === folderPath.value)
    .sort((a, b) => b.startedAt - a.startedAt)
)

// 历史任务（排除最新一条）
const latestId = computed(() => folderTasks.value[0]?.id)
const historyTasks = computed<Task[]>(() =>
  folderTasks.value.filter((t) => t.id !== latestId.value)
)

// 分页：每页 10 条历史
const {page, pageSize, pageSizes, pagedItems, total} = usePagination(historyTasks)

// 文件夹显示名（取最后一段）
const folderName = computed(() => {
  const parts = folderPath.value.split(/[\\/]/).filter(Boolean)
  return parts[parts.length - 1] || folderPath.value
})

const pathDisplay = computed(() => folderPath.value)

function back() {
  // 历史记录子页面 → 回到 /history 总览
  router.push({name: 'history'})
}

function openFolder() {
  import('@/api').then(({api}) => api.ShowItemInFolder(folderPath.value))
}

function viewTask(id: string) {
  // from=history-folder 让 ReportView 的返回按钮精确回到本子页面
  router.push({
    name: 'report',
    params: {id},
    query: {from: 'history-folder', path: encodeURIComponent(folderPath.value)}
  })
}

async function cancelTask(id: string) {
  await tasks.cancel(id)
  message.info('取消请求已发送')
}

async function removeTask(id: string) {
  await tasks.remove(id)
}
</script>

<template>
  <AppShell>
    <div class="history-page">
      <!-- 顶部：返回 + 文件夹路径（不要"历史记录"大标题，避免与顶栏标题重复） -->
      <div class="page-header">
        <NButton text @click="back" class="back-btn">
          <template #icon>
            <NIcon :component="ArrowBackOutline" />
          </template>
          返回历史记录
        </NButton>
        <NSpace align="center" :size="6" class="path-row">
          <NIcon :component="FolderOpenOutline" :size="14" color="#2080f0" />
          <NTooltip placement="top" trigger="hover">
            <template #trigger>
              <NText depth="3" class="folder-path">{{ pathDisplay }}</NText>
            </template>
            {{ pathDisplay }}
          </NTooltip>
          <NButton size="tiny" tertiary @click="openFolder">打开</NButton>
        </NSpace>
      </div>

      <!-- 历史记录列表（无 tab —— 与任务管理布局完全不同） -->
      <NEmpty v-if="historyTasks.length === 0" description="该文件夹暂无历史记录" style="margin-top: 40px" />
      <template v-else>
        <div class="history-list">
          <TaskCard
            v-for="t in pagedItems"
            :key="t.id"
            :task="t"
            hide-progress
            @view="viewTask"
            @report="viewTask"
            @cancel-task="cancelTask"
            @removed="removeTask"
          />
        </div>
        <!-- 列表分页器：条数 > 页大小时才显示 -->
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
    </div>
  </AppShell>
</template>

<style scoped>
/* 让 history-page 占据 content 区域全部高度，
   并把「顶部区」「列表区」「分页区」拆成三段独立的 flex 子项：
     - page-header → flex-shrink:0，始终贴顶、不滚动
     - history-list → flex:1 + overflow-y:auto，只有它滚动
     - list-pager  → flex-shrink:0，始终贴底、不参与滚动
   父级 .content 的 overflow:auto 在内容不超出时不会触发滚动条，
   所以视觉上只看到列表内部滚动。 */
.history-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding-top: 4px;
}
.page-header {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 4px 2px 12px;
  border-bottom: 1px solid var(--n-border-color);
  background: var(--n-card-color);
  margin: -4px -24px 0;
  padding-left: 24px;
  padding-right: 24px;
}
.back-btn {
  align-self: flex-start;
  font-size: 13px;
}
.path-row {
  font-size: 12px;
}
.folder-path {
  font-size: 12px;
  max-width: 720px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.history-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-top: 12px;
}
.list-pager {
  flex-shrink: 0;
  display: flex;
  justify-content: center;
  padding: 12px 0 8px;
  border-top: 1px solid var(--n-border-color);
  background: var(--n-card-color);
}
</style>