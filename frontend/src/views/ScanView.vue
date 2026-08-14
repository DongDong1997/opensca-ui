<script setup lang="ts">
import {ref, computed, onMounted} from 'vue'
import {useRoute, useRouter} from 'vue-router'
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

const scanPath = ref('')
const label = ref('')
const useCloud = ref(false)
const useLocalDB = ref(false)

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
    message.warning('请先选择项目目录')
    return
  }
  try {
    const id = await tasks.start({path: scanPath.value.trim(), label: label.value.trim() || undefined})
    message.success('扫描任务已创建')
    router.push({name: 'report', params: {id}})
  } catch (e) {
    message.error(`启动失败: ${String(e)}`)
  }
}
</script>


<template>
  <AppShell>
    <div class="scan-page">
      <NCard title="新建扫描">
        <DropZone @selected="onSelected" />
        <div class="scan-form">
          <NSpace vertical :size="12">
            <div>
              <NText strong>项目名</NText>
              <NTooltip placement="top" trigger="hover">
                <template #trigger>
                  <NInput
                    :value="projectName"
                    placeholder="选择路径后自动从文件夹名生成"
                    readonly
                    disabled
                    style="margin-top: 4px"
                  />
                </template>
                项目名绑定该扫描所属项目，作为历史记录分组的依据。
                取自所选路径的最末段文件夹名（或压缩包去扩展名），无法编辑。
              </NTooltip>
            </div>
            <div>
              <NText strong>任务标签（备注）</NText>
              <NInput
                v-model:value="label"
                placeholder="可选，例如：重构前扫描 / 回归验证"
                style="margin-top: 4px"
              />
            </div>
            <div>
              <NText strong>目标路径</NText>
              <NInput v-model:value="scanPath" placeholder="C:\\path\\to\\project" style="margin-top: 4px" />
            </div>
            <NSpace>
              <NButton type="primary" size="large" :disabled="!scanPath" @click="onStart">
                开始扫描
              </NButton>
              <NButton @click="$router.push('/tasks')">查看任务</NButton>
              <NButton quaternary @click="$router.push('/settings')">
                Token / 漏洞库 在设置中
              </NButton>
            </NSpace>
          </NSpace>
        </div>
      </NCard>

      <NCard title="最近任务" style="margin-top: 16px">
        <NSpace v-if="tasks.list.length === 0" align="center">
          <NText depth="3">还没有任务，启动一次扫描试试看</NText>
        </NSpace>
        <NSpace v-else vertical :size="8">
          <div
            v-for="t in tasks.list.slice(0, 5)"
            :key="t.id"
            class="recent-task"
            @click="$router.push({name: 'report', params: {id: t.id}})"
          >
            <NTag
              :type="t.status === 'success' ? 'success' : t.status === 'failed' ? 'error' : t.status === 'running' ? 'info' : 'default'"
              size="small"
            >
              {{ t.status }}
            </NTag>
            <NText strong>{{ t.projectName || t.label }}</NText>
            <NText v-if="t.label" depth="3" style="font-size: 12px">— {{ t.label }}</NText>
            <NText depth="3" style="font-size: 12px; margin-left: auto">
              {{ new Date(t.startedAt).toLocaleString() }}
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