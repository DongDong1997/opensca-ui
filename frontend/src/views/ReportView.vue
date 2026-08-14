<script setup lang="ts">
import {computed, onMounted, ref, watch} from 'vue'
import {useRouter, useRoute} from 'vue-router'
import {useI18n} from 'vue-i18n'
import {NTabs, NTabPane, NCard, NButton, NSpace, NText, NEmpty, NSpin, NTag, NAlert, NPagination, useMessage} from 'naive-ui'
import {FolderOpenOutline, RefreshOutline, DownloadOutline} from '@vicons/ionicons5'
import AppShell from '@/components/AppShell.vue'
import StatTiles from '@/components/StatTiles.vue'
import VulnGroupTable from '@/components/VulnGroupTable.vue'
import VulnDetailDrawer from '@/components/VulnDetailDrawer.vue'
import LogViewer from '@/components/LogViewer.vue'
import {useTasksStore} from '@/stores/tasks'
import {useTaskStream} from '@/composables/useTaskStream'
import {usePagination} from '@/composables/usePagination'
import {api} from '@/api'
import type {Vuln, Report, Task} from '@/api/types'

const props = defineProps<{id: string}>()
const router = useRouter()
const route = useRoute()
const tasks = useTasksStore()
const message = useMessage()
const {t} = useI18n()

const task = ref<Task | null>(null)
const report = ref<Report | null>(null)
const loading = ref(true)
const drawerShow = ref(false)
const selectedVuln = ref<Vuln | null>(null)

// 顶栏任务状态 tag 的翻译（数据侧是英文 status 枚举）
const STATUS_KEYS: Record<string, string> = {
  pending: 'task.status.pending',
  running: 'task.status.running',
  success: 'task.status.success',
  failed: 'task.status.failed',
  canceled: 'task.status.canceled'
}
const taskStatusLabel = computed(() => {
  if (!task.value) return ''
  return t(STATUS_KEYS[task.value.status] || 'task.status.pending')
})

const currentId = ref(props.id)
useTaskStream(() => currentId.value)

async function load(id: string) {
  loading.value = true
  currentId.value = id
  const detail = await tasks.fetchDetail(id)
  task.value = detail
  if (detail?.status === 'success' && detail.reportPath) {
    try {
      const r: Report = await api.GetTaskResult(id)
      report.value = r
    } catch (e) {
      report.value = null
      message.warning(t('report.parseFailed'))
    }
  } else {
    report.value = null
  }
  loading.value = false
}

onMounted(() => load(props.id))
watch(() => props.id, (id) => load(id))

watch(
  () => task.value?.status,
  (s) => {
    if (s === 'success' && task.value?.reportPath) {
      api.GetTaskResult(task.value.id).then((r: Report) => (report.value = r))
    }
  }
)

function onVulnClick(v: Vuln) {
  selectedVuln.value = v
  drawerShow.value = true
}

async function openReportFolder() {
  if (!task.value?.reportPath) {
    message.warning(t('report.noReportPath'))
    return
  }
  try {
    await api.ShowItemInFolder(task.value.reportPath)
  } catch (e) {
    message.error(t('common.openFailed', {msg: String(e)}))
  }
}

async function openHtmlReport() {
  if (!task.value?.htmlPath) {
    message.warning(t('report.htmlNotGenerated'))
    return
  }
  try {
    // 在系统浏览器里打开 HTML 报告（比资源管理器少一步操作）
    await api.OpenInFolder(toFileURL(task.value.htmlPath))
  } catch (e) {
    message.error(t('common.openFailed', {msg: String(e)}))
  }
}

// 把绝对路径转成 file:// URL（OpenInFolder 期望 URL 形式）。
function toFileURL(path: string) {
  let p = path.replace(/\\/g, '/')
  if (!p.startsWith('/')) p = '/' + p
  return 'file://' + encodeURI(p)
}

async function refresh() {
  await load(props.id)
}

const logs = computed(() => tasks.logs[props.id] || [])

// 组件依赖 tab 分页：组件数可能上百（Node monorepo / 大型 Java 工程），一次性渲染容易卡
const componentsSource = computed(() => report.value?.components ?? [])
const {page, pageSize, pageSizes, pagedItems: pagedComponents, total: componentsTotal} =
  usePagination(componentsSource)

function back() {
  // 根据来源页面决定返回目标：
  //   from=history-folder  → 单文件夹历史子页面（/history/:path）
  //   from=history-all     → 全部历史（顶栏"历史记录"）
  //   from=tasks-running   → 任务管理 - 运行中
  //   from=tasks-finished  → 任务管理 - 已完成
  //   其余（默认 from=tasks） → 任务管理（按项目）
  const from = (route.query.from as string | undefined) ?? 'tasks'
  if (from === 'history-folder' && typeof route.query.path === 'string' && route.query.path) {
    router.push({name: 'history-folder', params: {path: route.query.path}})
    return
  }
  if (from === 'history-all') {
    router.push({name: 'history'})
    return
  }
  if (from === 'tasks-running') {
    router.push('/tasks/running')
    return
  }
  if (from === 'tasks-finished') {
    router.push('/tasks/finished')
    return
  }
  router.push('/tasks')
}

// 返回按钮文案也跟来源走
const backLabel = computed(() => {
  const from = (route.query.from as string | undefined) ?? 'tasks'
  if (from === 'history-folder' || from === 'history-all') return t('report.backHistory')
  if (from === 'tasks-running') return t('report.backRunning')
  if (from === 'tasks-finished') return t('report.backFinished')
  return t('report.backTasks')
})
</script>

<template>
  <AppShell>
    <NSpace vertical :size="16">
      <div class="topbar">
        <NSpace align="center">
          <NButton @click="back">{{ backLabel }}</NButton>
          <NText strong style="font-size: 18px">{{ task?.label || t('common.loading') }}</NText>
          <NTag v-if="task" :type="task.status === 'success' ? 'success' : task.status === 'failed' ? 'error' : 'info'" size="small">
            {{ taskStatusLabel }}
          </NTag>
        </NSpace>
        <NSpace>
          <NButton @click="refresh" :loading="loading">
            <template #icon><span>↻</span></template>
            {{ t('report.refresh') }}
          </NButton>
          <NButton @click="openReportFolder">
            <template #icon><span>📁</span></template>
            {{ t('report.openReportDir') }}
          </NButton>
          <NButton type="primary" :disabled="!task?.htmlPath" @click="openHtmlReport">
            <template #icon><span>🌐</span></template>
            {{ t('report.viewInBrowser') }}
          </NButton>
        </NSpace>
      </div>

      <NSpin :show="loading">
        <!--
          CLI 透传的 task_info 警告（v3.x 常见：未配置漏洞库）。
          展示出来让用户知道为什么漏洞数为 0，而不是误以为 UI/解析出问题。
        -->
        <NAlert
          v-if="report?.warning"
          type="warning"
          :title="t('report.cliWarningTitle')"
          style="margin-bottom: 16px"
        >
          {{ report.warning }}
          <div style="margin-top: 4px; font-size: 12px; opacity: 0.85">
            {{ t('report.cliWarningHint') }}
          </div>
        </NAlert>

        <StatTiles :report="report" />

        <NCard style="margin-top: 16px">
          <NTabs type="line" animated default-value="vulns">
            <NTabPane name="vulns" :tab="t('report.tabVulns')">
              <NEmpty v-if="!report" :description="t('report.noReport')" style="margin: 60px 0" />
              <VulnGroupTable
                v-else
                :components="report.components.filter((c) => c.vulns.length > 0)"
                @row-click="onVulnClick"
              />
            </NTabPane>
            <NTabPane name="components" :tab="t('report.tabComponents')">
              <NEmpty v-if="!report" :description="t('report.noReportSimple')" style="margin: 60px 0" />
              <div v-else>
                <NText>{{ t('report.totalComponents', {n: report.totalComponents}) }}</NText>
                <NCard v-for="c in pagedComponents" :key="c.name + c.version" size="small" style="margin-top: 8px" :title="c.name">
                  <template #header-extra>
                    <NTag size="tiny">{{ c.language }}</NTag>
                  </template>
                  <NText code style="font-size: 12px">{{ c.version }}</NText>
                  <div style="margin-top: 8px; font-size: 12px; color: var(--n-text-color-3)">PURL: {{ c.purl }}</div>
                  <div v-if="c.vulns.length > 0" style="margin-top: 8px">
                    <NText depth="3" style="font-size: 12px">{{ t('report.vulnCountShort', {n: c.vulns.length}) }}</NText>
                  </div>
                </NCard>
                <!-- 组件依赖分页器：组件数 > 页大小时才显示 -->
                <div v-if="componentsTotal > pageSize" class="components-pager">
                  <NPagination
                    v-model:page="page"
                    :page-size="pageSize"
                    :item-count="componentsTotal"
                    :page-sizes="pageSizes"
                    show-size-picker
                    size="small"
                  />
                </div>
              </div>
            </NTabPane>
            <NTabPane name="logs" :tab="t('report.tabLogs')">
              <LogViewer :logs="logs" />
            </NTabPane>
            <NTabPane name="raw" :tab="t('report.tabRaw')">
              <pre style="background: var(--n-card-color); padding: 16px; border-radius: 6px; overflow: auto; max-height: 600px; font-size: 12px">{{ report ? JSON.stringify(report, null, 2) : '—' }}</pre>
            </NTabPane>
          </NTabs>
        </NCard>
      </NSpin>
    </NSpace>

    <VulnDetailDrawer v-model:show="drawerShow" :vuln="selectedVuln" />
  </AppShell>
</template>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.components-pager {
  display: flex;
  justify-content: center;
  padding: 16px 0 8px;
  margin-top: 12px;
}
</style>
