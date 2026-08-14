<script setup lang="ts">
import {computed, onMounted, ref, watch} from 'vue'
import {useRouter} from 'vue-router'
import {
  NLayout,
  NLayoutHeader,
  NLayoutContent,
  NIcon,
  NText,
  NSpace,
  NButton,
  NTag,
  NEmpty,
  NPopconfirm,
  NTooltip,
  NPagination,
  useMessage
} from 'naive-ui'
import {
  SettingsOutline,
  ShieldCheckmarkOutline,
  FolderOutline,
  TimeOutline,
  TrashOutline,
  FolderOpenOutline,
  RocketOutline
} from '@vicons/ionicons5'
import {useRecentStore} from '@/stores/recent'
import {useConfigStore} from '@/stores/config'
import {api} from '@/api'

const router = useRouter()
const recent = useRecentStore()
const cfg = useConfigStore()
const message = useMessage()

// 软编码版本号：与 SettingsView.vue 关于模块保持一致，未来可换成构建时常量。
const APP_VERSION = 'v0.1.0'

// 每个项目的本地扫描数（path → count）
const projectScanCounts = ref<Record<string, number>>({})

// 最近项目列表分页
const PAGE_SIZE = 8
const currentPage = ref(1)
const pagedRecent = computed(() => {
  const start = (currentPage.value - 1) * PAGE_SIZE
  return recent.list.slice(start, start + PAGE_SIZE)
})
// 列表长度变化（比如新增/移除/清空）时，自动回到第 1 页，避免停在不存在的页
watch(
  () => recent.list.length,
  () => {
    currentPage.value = 1
  }
)

async function loadProjectCounts() {
  const counts: Record<string, number> = {}
  await Promise.all(
    recent.list.map(async (r) => {
      try {
        const list = await api.GetProjectScanHistory(r.path)
        counts[r.path] = list.length
      } catch {
        counts[r.path] = 0
      }
    })
  )
  projectScanCounts.value = counts
}

function formatTime(ts: number) {
  if (!ts) return ''
  const d = new Date(ts)
  const now = Date.now()
  const diff = now - ts
  // 一分钟内
  if (diff < 60_000) return '刚刚'
  // 一小时内
  if (diff < 3600_000) return `${Math.floor(diff / 60_000)} 分钟前`
  // 一天内
  if (diff < 86400_000) return `${Math.floor(diff / 3600_000)} 小时前`
  // 一周内
  if (diff < 7 * 86400_000) return `${Math.floor(diff / 86400_000)} 天前`
  // 否则展示日期
  return d.toLocaleDateString()
}

function displayLabel(r: {path: string; label: string; projectName?: string}) {
  // 优先用稳定的 projectName（目录 basename / 压缩包去扩展名）；
  // 没有（老版本 recent.json 没这个字段）再退化到 label / path basename。
  if (r.projectName && r.projectName.trim()) return r.projectName
  if (r.label && r.label.trim()) return r.label
  return r.path.split(/[\\/]/).pop() || r.path
}

function openProject(path: string, label: string) {
  router.push({
    name: 'scan',
    query: {
      path: encodeURIComponent(path),
      label: label ? encodeURIComponent(label) : ''
    }
  })
}

async function openProjectFolder(path: string) {
  try {
    await api.OpenProjectFolder(path)
  } catch (e) {
    message.error(`打开失败: ${String(e)}`)
  }
}

async function removeOne(path: string) {
  await recent.remove(path)
  message.success('已从最近列表移除')
}

async function clearAll() {
  await recent.clear()
  message.success('最近列表已清空')
}

async function quickNewScan() {
  // 跳到新建扫描时不带路径，让用户自己选
  router.push('/scan')
}

onMounted(async () => {
  await recent.refresh()
  await loadProjectCounts()
})
</script>


<template>
  <NLayout position="absolute" class="home-shell">
    <!-- 顶部：品牌 + 版本号 -->
    <NLayoutHeader bordered class="home-header">
      <div class="brand">
        <NIcon :size="28" :component="ShieldCheckmarkOutline" color="#18a058" />
        <span class="brand-text">OpenSCA UI</span>
      </div>
      <div class="corner">
        <NTag :type="cfg.cliValid ? 'success' : 'warning'" size="small" round>
          {{ APP_VERSION }}
        </NTag>
      </div>
    </NLayoutHeader>

    <!-- 内容 -->
    <NLayoutContent class="home-content" :native-scrollbar="false">
      <div class="home-main">
        <!-- 居中标题区（ComfyUI 风格：shield 图标 + 大标题 + 副标题 + 主操作） -->
        <section class="hero">
          <div class="hero-logo">
            <NIcon :size="96" :component="ShieldCheckmarkOutline" color="#18a058" />
          </div>
          <h1 class="hero-title">OpenSCA UI</h1>
          <p class="hero-subtitle">
            一站式开源软件成分分析与漏洞检测工具
          </p>
          <NSpace :size="12" style="margin-top: 28px">
            <NButton type="primary" size="large" round @click="quickNewScan">
              <template #icon>
                <NIcon :component="RocketOutline" />
              </template>
              新建扫描
            </NButton>
            <NButton size="large" round @click="router.push('/history')">
              <template #icon>
                <NIcon :component="TimeOutline" />
              </template>
              历史记录
            </NButton>
            <NButton size="large" round @click="router.push('/settings')">
              <template #icon>
                <NIcon :component="SettingsOutline" />
              </template>
              设置
            </NButton>
          </NSpace>
        </section>

        <!-- 最近项目 -->
        <div class="section">
          <div class="section-header">
            <NSpace align="center" :size="8">
              <NIcon :size="18" :component="TimeOutline" />
              <NText strong style="font-size: 15px">最近打开的项目</NText>
              <NTag v-if="recent.list.length" size="small" round>
                {{ recent.list.length }}
              </NTag>
            </NSpace>
            <NPopconfirm
              v-if="recent.list.length"
              @positive-click="clearAll"
              positive-text="清空"
              negative-text="取消"
            >
              <template #trigger>
                <NButton quaternary size="small" type="error">
                  <template #icon>
                    <NIcon :component="TrashOutline" />
                  </template>
                  清空
                </NButton>
              </template>
              确认清空所有最近项目？此操作不可撤销。
            </NPopconfirm>
          </div>

          <NEmpty
            v-if="!recent.loading && recent.list.length === 0"
            description="还没有扫描过的项目"
            style="padding: 48px 0"
          >
            <template #extra>
              <NText depth="3" style="font-size: 13px">
                开始一次扫描后，最近的项目会自动出现在这里
              </NText>
            </template>
          </NEmpty>

          <div v-else class="recent-list">
            <div
              v-for="r in pagedRecent"
              :key="r.path"
              class="recent-item"
              tabindex="0"
              role="button"
              @click="openProject(r.path, r.label)"
              @keydown.enter="openProject(r.path, r.label)"
            >
              <NIcon :size="22" :component="FolderOutline" color="#2080f0" />
              <div class="recent-info">
                <div class="recent-label">
                  <NText strong>{{ displayLabel(r) }}</NText>
                  <NTag v-if="r.useCount > 1" size="tiny" round style="margin-left: 8px">
                    扫描过 {{ r.useCount }} 次
                  </NTag>
                  <NTag
                    v-if="projectScanCounts[r.path] > 0"
                    size="tiny"
                    round
                    type="info"
                    style="margin-left: 8px"
                  >
                    <template #icon>
                      <NIcon :component="FolderOpenOutline" />
                    </template>
                    已存档 {{ projectScanCounts[r.path] }} 条
                  </NTag>
                </div>
                <NText depth="3" class="recent-path">{{ r.path }}</NText>
              </div>
              <NText depth="3" class="recent-time" style="font-size: 12px">
                {{ formatTime(r.lastAt) }}
              </NText>
              <NTooltip>
                <template #trigger>
                  <NButton
                    quaternary
                    circle
                    size="small"
                    @click.stop="openProjectFolder(r.path)"
                    title="打开项目目录"
                  >
                    <template #icon>
                      <NIcon :component="FolderOpenOutline" />
                    </template>
                  </NButton>
                </template>
                打开项目目录
              </NTooltip>
              <NPopconfirm
                @positive-click="removeOne(r.path)"
                @click.stop
                positive-text="移除"
                negative-text="取消"
              >
                <template #trigger>
                  <NButton
                    quaternary
                    circle
                    size="small"
                    @click.stop
                    title="从最近列表移除"
                  >
                    <template #icon>
                      <NIcon :component="TrashOutline" />
                    </template>
                  </NButton>
                </template>
                从最近列表移除该路径？
              </NPopconfirm>
            </div>
          </div>

          <!-- 分页器：只展示一页装不下时才需要 -->
          <div v-if="recent.list.length > PAGE_SIZE" class="recent-pager">
            <NPagination
              v-model:page="currentPage"
              :page-count="Math.ceil(recent.list.length / PAGE_SIZE)"
              :page-size="PAGE_SIZE"
              show-quick-jumper
            />
          </div>
        </div>
      </div>
    </NLayoutContent>
  </NLayout>
</template>

<style scoped>
.home-shell {
  height: 100vh;
  background: var(--n-color);
}
.home-header {
  height: 64px;
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--n-card-color);
}
.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}
.brand-text {
  font-size: 20px;
  font-weight: 600;
}
.corner {
  display: flex;
  align-items: center;
  gap: 8px;
}
.home-content {
  padding: 24px;
  height: calc(100vh - 64px);
  /* 主页不要滚动条：让内容在视口内自适应展示，
     整体超出时允许由外层布局自然处理。 */
  overflow: visible;
}
.home-main {
  max-width: 960px;
  margin: 0 auto;
}

/* ComfyUI 风格 hero：居中 logo + 大标题 + 副标题 + 操作按钮组 */
.hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 56px 24px 48px;
  margin-bottom: 24px;
}
.hero-logo {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20px;
  filter: drop-shadow(0 4px 16px rgba(24, 160, 88, 0.25));
}
.hero-title {
  font-size: 56px;
  font-weight: 700;
  line-height: 1.1;
  letter-spacing: -1.5px;
  margin: 0;
  background: linear-gradient(135deg, #18a058 0%, #2080f0 100%);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
  color: transparent;
}
.hero-subtitle {
  font-size: 16px;
  line-height: 1.6;
  color: var(--n-text-color-2);
  margin: 12px 0 0;
  max-width: 560px;
}

.section {
  background: var(--n-card-color);
  border: 1px solid var(--n-border-color);
  border-radius: 8px;
  padding: 20px;
}
.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--n-border-color);
}
.recent-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.recent-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s ease;
  outline: none;
}
.recent-item:hover {
  background: var(--n-action-color);
}
.recent-item:focus-visible {
  outline: 2px solid var(--n-primary-color);
  outline-offset: -2px;
}
.recent-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.recent-label {
  display: flex;
  align-items: center;
  font-size: 14px;
}
.recent-path {
  font-size: 12px;
  word-break: break-all;
  font-family: var(--n-font-family-mono);
}
.recent-time {
  flex-shrink: 0;
  white-space: nowrap;
}
.recent-pager {
  display: flex;
  justify-content: center;
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--n-border-color);
}
</style>