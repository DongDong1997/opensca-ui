<script setup lang="ts">
import {ref, computed, onMounted} from 'vue'
import {useRoute, useRouter} from 'vue-router'
import {useI18n} from 'vue-i18n'
import {NCard, NButton, NSpace, NText, NTag, NInput, NTooltip, useMessage} from 'naive-ui'
import DropZone from '@/components/DropZone.vue'
import AppShell from '@/components/AppShell.vue'
import {useTasksStore} from '@/stores/tasks'
import {useConfigStore} from '@/stores/config'

const router = useRouter()
const route = useRoute()
const tasks = useTasksStore()
const cfg = useConfigStore()
const message = useMessage()
const {t} = useI18n()

const scanPath = ref('')
const label = ref('')
const useCloud = ref(false)
const useLocalDB = ref(false)

// 最近任务列表里状态 tag 的翻译（数据侧是英文 status 枚举）
const STATUS_KEYS: Record<string, string> = {
  pending: 'task.status.pending',
  running: 'task.status.running',
  success: 'task.status.success',
  failed: 'task.status.failed',
  canceled: 'task.status.canceled'
}
function statusLabel(s: string) {
  return t(STATUS_KEYS[s] || 'task.status.pending')
}

// 项目名：从路径直接推导，固定不可编辑。
// 规则与后端 Manager.deriveProjectName 保持一致：
//   - 目录 → basename
//   - zip / tar.gz 等压缩包 → basename 去扩展名（兼容 .tar.gz 双扩展）
const projectName = computed(() => {
  const p = scanPath.value.trim()
  if (!p) return ''
  const isArchive = /\.(zip|tar\.gz|tgz)$/i.test(p)
  const base = p.split(/[\\/]/).filter(Boolean).pop() ?? p
  if (!base || base === '.' || base === '/') return ''
  if (!isArchive) return base
  return base.replace(/\.(zip|tar\.gz|tgz)$/i, '')
})

onMounted(async () => {
  await tasks.refresh()
  // 从首页点击"最近项目"进来时，URL 带 ?path=&label=
  // 这里只读一次，刷新页面再次导航会被新的 query 覆盖
  const q = route.query
  if (q.path) {
    try {
      scanPath.value = decodeURIComponent(String(q.path))
    } catch {
      scanPath.value = String(q.path)
    }
  }
  if (q.label) {
    try {
      label.value = decodeURIComponent(String(q.label))
    } catch {
      label.value = String(q.label)
    }
  }
})

async function onSelected(path: string) {
  scanPath.value = path
}

async function onStart() {
  if (!scanPath.value.trim()) {
    message.warning(t('scan.selectFirst'))
    return
  }
  try {
    const id = await tasks.start({path: scanPath.value.trim(), label: label.value.trim() || undefined})
    message.success(t('scan.taskCreated'))
    router.push({name: 'report', params: {id}})
  } catch (e) {
    message.error(t('common.startFailed', {msg: String(e)}))
  }
}
</script>


<template>
  <AppShell>
    <div class="scan-page">
      <NCard :title="t('scan.title')">
        <DropZone @selected="onSelected" />
        <div class="scan-form">
          <NSpace vertical :size="12">
            <div>
              <NText strong>{{ t('scan.projectName') }}</NText>
              <NTooltip placement="top" trigger="hover">
                <template #trigger>
                  <NInput
                    :value="projectName"
                    :placeholder="t('scan.projectNamePlaceholder')"
                    readonly
                    disabled
                    style="margin-top: 4px"
                  />
                </template>
                {{ t('scan.projectNameTooltip') }}
              </NTooltip>
            </div>
            <div>
              <NText strong>{{ t('scan.taskLabel') }}</NText>
              <NInput
                v-model:value="label"
                :placeholder="t('scan.labelPlaceholder')"
                style="margin-top: 4px"
              />
            </div>
            <div>
              <NText strong>{{ t('scan.targetPath') }}</NText>
              <NInput v-model:value="scanPath" placeholder="C:\\path\\to\\project" style="margin-top: 4px" />
            </div>
            <NSpace>
              <NButton type="primary" size="large" :disabled="!scanPath" @click="onStart">
                {{ t('scan.start') }}
              </NButton>
              <NButton @click="$router.push('/tasks')">{{ t('scan.viewTasks') }}</NButton>
              <NButton quaternary @click="$router.push('/settings')">
                {{ t('scan.tokenHint') }}
              </NButton>
            </NSpace>
          </NSpace>
        </div>
      </NCard>

      <NCard :title="t('scan.recentTasks')" style="margin-top: 16px">
        <NSpace v-if="tasks.list.length === 0" align="center">
          <NText depth="3">{{ t('scan.noTasks') }}</NText>
        </NSpace>
        <NSpace v-else vertical :size="8">
          <div
            v-for="task in tasks.list.slice(0, 5)"
            :key="task.id"
            class="recent-task"
            @click="$router.push({name: 'report', params: {id: task.id}})"
          >
            <NTag
              :type="task.status === 'success' ? 'success' : task.status === 'failed' ? 'error' : task.status === 'running' ? 'info' : 'default'"
              size="small"
            >
              {{ statusLabel(task.status) }}
            </NTag>
            <NText strong>{{ task.projectName || task.label }}</NText>
            <NText v-if="task.label" depth="3" style="font-size: 12px">— {{ task.label }}</NText>
            <NText depth="3" style="font-size: 12px; margin-left: auto">
              {{ new Date(task.startedAt).toLocaleString() }}
            </NText>
          </div>
        </NSpace>
      </NCard>
    </div>
  </AppShell>
</template>

<style scoped>
.scan-page {
  max-width: 960px;
  margin: 0 auto;
}
.scan-form {
  margin-top: 24px;
}
.opts {
  padding: 4px 0;
}
.recent-task {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
}
.recent-task:hover {
  background: var(--n-action-color);
}
</style>
