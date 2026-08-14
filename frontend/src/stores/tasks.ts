import {defineStore} from 'pinia'
import {computed, ref} from 'vue'
import {api} from '@/api'
import type {Task, TaskStatus, TaskSummary} from '@/api/types'

/**
 * 日志条目（含时间戳）。任务详情页按 ts 排序展示。
 */
export interface LogEntry {
  ts: number
  line: string
}

export const useTasksStore = defineStore('tasks', () => {
  // byId 存详情（含日志），order 为最新在前
  const byId = ref<Record<string, Task>>({})
  const logs = ref<Record<string, LogEntry[]>>({})
  const order = ref<string[]>([])
  const selectedId = ref<string | null>(null)
  const loading = ref(false)

  // 当前会话的起点。后端 ListTasks 会把"内存 + 历史"合并返回，
  // 我们用 sessionStart 当作"本次会话内"的时间边界：
  //   startedAt >= sessionStart  → 本次会话产生的任务（任务管理页可见）
  //   startedAt <  sessionStart  → 历史持久化任务（只在历史记录页可见）
  const sessionStart = Date.now()

  // 把任意来源的时间戳归一化成 UnixMilli 数字。
  // 后端通过 Wails 序列化 Go time.Time 时给的是 RFC3339 字符串
  // （例如 "2026-08-12T06:30:45.123Z"），前端声明却是 number。
  // 不归一化会导致 startedAt >= sessionStart 这种数字比较把字符串转 NaN → false，
  // 新任务会被 currentSessionList 错误地过滤掉。
  function toUnixMilli(v: unknown): number {
    if (typeof v === 'number') return Number.isFinite(v) ? v : 0
    if (typeof v === 'string') {
      if (!v) return 0
      const t = Date.parse(v)
      return Number.isNaN(t) ? 0 : t
    }
    return 0
  }

  const list = computed<Task[]>(() =>
    order.value.map((id) => byId.value[id]).filter(Boolean)
  )
  // 任务管理专用：仅本次会话的任务
  const currentSessionList = computed<Task[]>(() =>
    list.value.filter((t) => t.startedAt >= sessionStart)
  )
  const running = computed(() => list.value.filter((t) => t.status === 'running'))
  const finished = computed(() =>
    list.value.filter((t) => t.status === 'success' || t.status === 'failed' || t.status === 'canceled')
  )
  const selected = computed<Task | null>(() =>
    selectedId.value ? byId.value[selectedId.value] ?? null : null
  )

  function upsert(task: Task) {
    // 先归一化时间字段：后端可能给 RFC3339 字符串，local start() 给数字，统一成 number。
    const normalized: Task = {
      ...task,
      startedAt: toUnixMilli(task.startedAt),
      finishedAt: toUnixMilli(task.finishedAt)
    }
    if (!byId.value[normalized.id]) {
      order.value.unshift(normalized.id)
    }
    byId.value[normalized.id] = {...byId.value[normalized.id], ...normalized}
  }

  function appendLog(taskID: string, line: string, ts?: number) {
    if (!logs.value[taskID]) logs.value[taskID] = []
    logs.value[taskID].push({ts: ts ?? Date.now(), line})
    // 限制前端内存中日志条数，超出截断最早
    const MAX = 5000
    if (logs.value[taskID].length > MAX) {
      logs.value[taskID] = logs.value[taskID].slice(-MAX)
    }
  }

  function setProgress(taskID: string, percent: number, stage: string) {
    if (!byId.value[taskID]) return
    byId.value[taskID].progress = percent
    byId.value[taskID].stage = stage
  }

  function setStatus(taskID: string, status: TaskStatus) {
    if (!byId.value[taskID]) return
    byId.value[taskID].status = status
    // 终态时把任务移到列表末尾
    if (status === 'success' || status === 'failed' || status === 'canceled') {
      order.value = [taskID, ...order.value.filter((id) => id !== taskID)]
    }
  }

  function setFinished(taskID: string, durationMs: number, reportPath: string) {
    if (!byId.value[taskID]) return
    byId.value[taskID].durationMs = durationMs
    byId.value[taskID].finishedAt = Date.now()
    byId.value[taskID].reportPath = reportPath
  }

  function select(id: string | null) {
    selectedId.value = id
  }

  async function refresh() {
    loading.value = true
    try {
      const summaries = await api.ListTasks()
      summaries.forEach((s: TaskSummary) => {
        const existing = byId.value[s.id]
        upsert({
          ...(existing ?? ({} as Task)),
          id: s.id,
          label: s.label,
          path: s.path,
          status: s.status,
          progress: s.progress,
          startedAt: s.startedAt,
          finishedAt: s.finishedAt,
          durationMs: s.durationMs
        } as Task)
      })
    } catch (e) {
      console.warn('ListTasks failed:', e)
    } finally {
      loading.value = false
    }
  }

  async function fetchDetail(id: string): Promise<Task | null> {
    try {
      const t = await api.GetTask(id)
      upsert(t)
      return t
    } catch (e) {
      console.warn('GetTask failed:', e)
      return null
    }
  }

  async function fetchLogs(id: string, offset = 0): Promise<string> {
    try {
      const blob = await api.GetTaskLogs(id, offset)
      return blob ?? ''
    } catch (e) {
      console.warn('GetTaskLogs failed:', e)
      return ''
    }
  }

  async function start(req: {path: string; label?: string}) {
    const id = await api.StartScan({path: req.path, label: req.label ?? ''})
    // 项目名与后端 deriveProjectName 保持一致：目录取 basename，压缩包去扩展名。
    const isArchive = /\.(zip|tar\.gz|tgz)$/i.test(req.path)
    const base = req.path.split(/[\\/]/).filter(Boolean).pop() ?? req.path
    const projectName = isArchive
      ? base.replace(/\.(zip|tar\.gz|tgz)$/i, '')
      : base
    upsert({
      id,
      label: (req.label ?? '').trim(),
      projectName,
      path: req.path,
      status: 'pending',
      progress: 0,
      stage: '等待中',
      startedAt: Date.now(),
      finishedAt: 0,
      durationMs: 0,
      exitCode: 0,
      error: '',
      reportPath: '',
      htmlPath: '',
      logPath: ''
    })
    return id
  }

  async function cancel(id: string) {
    try {
      await api.CancelScan(id)
    } catch (e) {
      console.warn('CancelScan failed:', e)
    }
  }

  async function remove(id: string) {
    try {
      await api.DeleteTask(id)
    } catch (e) {
      console.warn('DeleteTask failed:', e)
    }
    delete byId.value[id]
    delete logs.value[id]
    order.value = order.value.filter((x) => x !== id)
    if (selectedId.value === id) selectedId.value = null
  }

  return {
    byId,
    logs,
    order,
    selectedId,
    loading,
    sessionStart,
    list,
    currentSessionList,
    running,
    finished,
    selected,
    upsert,
    appendLog,
    setProgress,
    setStatus,
    setFinished,
    select,
    refresh,
    fetchDetail,
    fetchLogs,
    start,
    cancel,
    remove
  }
})