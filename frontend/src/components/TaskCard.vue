<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'
import {NCard, NTag, NProgress, NSpace, NButton, NButtonGroup, NIcon, NTooltip, NText} from 'naive-ui'
import {PlayCircleOutline, CloseCircleOutline, TrashOutline, DocumentTextOutline, FolderOpenOutline} from '@vicons/ionicons5'
import SeverityTag from './SeverityTag.vue'
import {api} from '@/api'
import type {Task, TaskStatus} from '@/api/types'

const props = defineProps<{
  task: Task
  showLog?: boolean
  /** 历史记录里进度条意义不大（任务都结束了），传 true 隐藏整个进度区 */
  hideProgress?: boolean
}>()
const emit = defineEmits<{
  (e: 'view', id: string): void
  (e: 'report', id: string): void
  (e: 'cancel-task', id: string): void
  (e: 'removed', id: string): void
}>()

const {t} = useI18n()

// 颜色 / 类型静态，label 随语言取 t()（computed 保证 locale 变化时重新求值）
const statusMap: Record<TaskStatus, {labelKey: string; type: 'default' | 'info' | 'success' | 'error' | 'warning'; color: string}> = {
  pending: {labelKey: 'task.status.pending', type: 'default', color: '#909399'},
  running: {labelKey: 'task.status.running', type: 'info', color: '#2080f0'},
  success: {labelKey: 'task.status.success', type: 'success', color: '#18a058'},
  failed: {labelKey: 'task.status.failed', type: 'error', color: '#d03050'},
  canceled: {labelKey: 'task.status.canceled', type: 'warning', color: '#f0a020'}
}

const s = computed(() => {
  const base = statusMap[props.task.status as TaskStatus]
  return {label: t(base.labelKey), type: base.type, color: base.color}
})
const isRunning = computed(() => props.task.status === 'running' || props.task.status === 'pending')
const isFinished = computed(
  () => props.task.status === 'success' || props.task.status === 'failed' || props.task.status === 'canceled'
)
const canReport = computed(() => props.task.status === 'success' && !!props.task.reportPath)

// 显示用：项目名优先，回退到路径 basename（历史数据兜底），再回退到 label
const displayProject = computed(
  () => props.task.projectName || props.task.path.split(/[\\/]/).pop() || props.task.label || ''
)

// 任务备注（label）作为可选副标题展示
const displayNote = computed(() => (props.task.label ?? '').trim())

// 时间格式：MM-DD HH:mm:ss（无年份，避免占位；同一会话内年份恒定）
function formatClock(ts: number): string {
  if (!ts || ts <= 0) return ''
  const d = new Date(ts)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

// 任务卡右下角的时间标签：运行中显示"开始于 ..."，已完成显示"完成于 ..."
const timeTag = computed(() => {
  if (isFinished.value && props.task.finishedAt > 0) {
    return t('taskcard.finishedAt', {time: formatClock(props.task.finishedAt)})
  }
  if (props.task.startedAt > 0) {
    return t('taskcard.startedAt', {time: formatClock(props.task.startedAt)})
  }
  return ''
})

// 进度条旁的 stage 文本：数据侧仅有的中文种子值"等待中"渲染时映射；其余后端 stage 原样透传
const displayStage = computed(() => {
  if (!props.task.stage) return isRunning.value ? t('common.scanning') : ''
  return props.task.stage === '等待中' ? t('task.status.pending') : props.task.stage
})

const durationText = computed(() =>
  props.task.durationMs > 0 ? t('taskcard.duration', {s: (props.task.durationMs / 1000).toFixed(1)}) : ''
)

function onCancel() {
  emit('view', props.task.id)
}
</script>

<template>
  <NCard size="small" hoverable class="task-card" @click="emit('view', task.id)">
    <div class="task-row">
      <div class="task-meta">
        <NSpace align="center" :size="8" :wrap="false">
          <NTag :type="s.type" size="small" round>
            <template #icon>
              <span class="status-dot" :style="{background: s.color}" />
            </template>
            {{ s.label }}
          </NTag>
          <!-- 项目名：作为记录的主标识 -->
          <NText strong>{{ displayProject }}</NText>
          <!-- 任务备注：仅在用户填了 label 时展示 -->
          <NText v-if="displayNote" depth="2" class="task-note">— {{ displayNote }}</NText>
        </NSpace>
        <!-- 二级行：路径 + 时间戳 -->
        <div class="task-sub">
          <NText depth="3" class="task-path">{{ task.path }}</NText>
          <NText v-if="timeTag" depth="3" class="task-time">· {{ timeTag }}</NText>
        </div>
      </div>
      <NSpace>
        <NButton v-if="isRunning" size="tiny" tertiary type="warning" @click.stop="$emit('cancel-task', task.id)">
          {{ t('taskcard.cancel') }}
        </NButton>
        <NButton v-if="canReport" size="tiny" tertiary type="primary" @click.stop="emit('report', task.id)">
          {{ t('taskcard.viewReport') }}
        </NButton>
        <NButton size="tiny" tertiary @click.stop="emit('removed', task.id)">{{ t('taskcard.delete') }}</NButton>
      </NSpace>
    </div>

    <div v-if="!hideProgress" class="task-progress">
      <NProgress
        :percentage="task.progress"
        :status="task.status === 'failed' ? 'error' : task.status === 'canceled' ? 'default' : task.status === 'success' ? 'success' : 'default'"
        :show-indicator="isRunning"
        :indicator-placement="'inside'"
        :height="6"
      />
      <NText depth="3" style="margin-left: 12px; font-size: 12px">
        {{ displayStage }}
        <span v-if="durationText"> · {{ durationText }}</span>
      </NText>
    </div>
  </NCard>
</template>

<style scoped>
.task-card {
  margin-bottom: 12px;
}
.task-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 8px;
}
.task-meta {
  flex: 1;
  min-width: 0;
}
.task-note {
  font-size: 13px;
  font-style: italic;
  flex-shrink: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.task-sub {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
  margin-left: 4px;
  font-size: 12px;
  min-width: 0;
}
.task-path {
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  flex: 1;
}
.task-time {
  font-size: 12px;
  font-family: var(--n-font-family-mono);
  white-space: nowrap;
  flex-shrink: 0;
}
.task-progress {
  display: flex;
  align-items: center;
}
.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 4px;
}
</style>
