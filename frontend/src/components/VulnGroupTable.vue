<script setup lang="ts">
import {computed, h, ref, watch} from 'vue'
import {
  NCollapse,
  NCollapseItem,
  NDataTable,
  NTag,
  NText,
  NEmpty,
  NSelect,
  NSpace,
  NInput,
  NTooltip,
  NPagination,
  type DataTableColumns,
} from 'naive-ui'
import SeverityTag from './SeverityTag.vue'
import type {Vuln, Severity} from '@/api/types'

const props = defineProps<{components: Array<{name: string; version: string; language: string; vulns: Vuln[]; licenses?: string[]; direct?: boolean}>}>()
const emit = defineEmits<{(e: 'row-click', v: Vuln): void}>()

const search = ref('')
const severityFilter = ref<Severity[]>([])
const expandedNames = ref<string[]>([]) // 默认全部收拢；用户点击展开

// 外层（组件列表）分页：一页 10 个组件，避免 43 个面板同时展开挤爆滚动区
const componentPage = ref(1)
const componentPageSize = ref(10)

const severityOptions = [
  {label: '严重', value: 'critical'},
  {label: '高危', value: 'high'},
  {label: '中危', value: 'medium'},
  {label: '低危', value: 'low'},
  {label: '提示', value: 'info'}
]

// 利用难度数字 → 中文标签
function exploitLevelLabel(n: number): string {
  switch (n) {
    case 1: return '容易'
    case 2: return '中等'
    case 3: return '困难'
    default: return '未知'
  }
}

// 严重度优先级（用于组内排序）
const SEV_RANK: Record<string, number> = {critical: 0, high: 1, medium: 2, low: 3, info: 4, unknown: 5}

// Naive UI v2.40 的 ellipsis.tooltip 类型只接受 boolean | PopoverProps，
// 不再支持函数形式。这里通过 PopoverProps 配 keepAliveOnHover（鼠标可移入气泡内部选中文字）；
// 气泡宽度由全局 .ellipsis-tooltip-popup 样式约束（见 <style scoped> 内的 :global 规则）。
// 当前 max-width: 160px ≈ 8 个中文字 / 16 个英文字符一行，约原默认宽度的 1/3。
const NARROW_TOOLTIP_PROPS = {
  placement: 'top',
  keepAliveOnHover: true,
  showArrow: true,
  // 注意：这版 Naive UI 的 prop 是 contentClass（不是 panelClassName），而且它直接
  // 加到 .n-popover__content 元素本身（不是祖先容器）。配合下方 :global 选择器命中。
  contentClass: 'ellipsis-tooltip-popup'
} as const

// 跨组过滤：返回新 components 数组；任何组过滤后为空就整组隐藏
const filteredComponents = computed(() => {
  const q = search.value.trim().toLowerCase()
  const sevSet = new Set(severityFilter.value as string[])
  return props.components
    .map((c) => {
      const vs = c.vulns.filter((v) => {
        if (sevSet.size > 0 && !sevSet.has(v.severity as string)) return false
        if (q) {
          const hit =
            v.title?.toLowerCase().includes(q) ||
            v.id?.toLowerCase().includes(q) ||
            v.cve?.some((x) => x.toLowerCase().includes(q)) ||
            v.cwe?.some((x) => x.toLowerCase().includes(q)) ||
            v.description?.toLowerCase().includes(q) ||
            v.suggestion?.toLowerCase().includes(q) ||
            (v.source?.toLowerCase().includes(q) ?? false)
          if (!hit) return false
        }
        return true
      })
      // 组内排序：严重度倒序 → 发布日期倒序 → id 升序
      vs.sort((a, b) => {
        const r = (SEV_RANK[a.severity] ?? 9) - (SEV_RANK[b.severity] ?? 9)
        if (r !== 0) return r
        if (a.releaseDate && b.releaseDate && a.releaseDate !== b.releaseDate) {
          return a.releaseDate < b.releaseDate ? 1 : -1
        }
        return (a.id || '').localeCompare(b.id || '')
      })
      return {...c, vulns: vs}
    })
    .filter((c) => c.vulns.length > 0)
})

// 跨组统计（用于顶部 "共 X 个组件 / Y 条漏洞"）
const totalVulnsShown = computed(() => filteredComponents.value.reduce((s, c) => s + c.vulns.length, 0))

// 外层分页：按 componentPageSize 切片
const pagedComponents = computed(() => {
  const start = (componentPage.value - 1) * componentPageSize.value
  return filteredComponents.value.slice(start, start + componentPageSize.value)
})

// 过滤条件变了 → 自动回到第 1 页，避免空页
watch([search, severityFilter], () => {
  componentPage.value = 1
})

// 每个组件的"折叠面板"内部表格列
const columns: DataTableColumns<Vuln> = [
  {
    title: '漏洞名称',
    key: 'title',
    width: 240,
    ellipsis: {tooltip: NARROW_TOOLTIP_PROPS},
    render: (row) => row.title || '-'
  },
  {
    title: '风险等级',
    key: 'severity',
    width: 90,
    render: (row) => h(SeverityTag, {severity: row.severity as Severity})
  },
  {
    title: '漏洞编号',
    key: 'id',
    width: 180,
    render: (row) => h(NText, {code: true, depth: 3}, () => row.id || '-')
  },
  {
    title: '发布日期',
    key: 'releaseDate',
    width: 110,
    render: (row) => row.releaseDate || '-'
  },
  {
    title: '利用难度',
    key: 'exploitLevel',
    width: 90,
    render: (row) => exploitLevelLabel(row.exploitLevel ?? 0)
  },
  {
    title: '攻击类型',
    key: 'attackType',
    width: 90,
    render: (row) => row.source || '-'
  },
  {
    title: '漏洞描述',
    key: 'description',
    minWidth: 280,
    ellipsis: {tooltip: NARROW_TOOLTIP_PROPS},
    render: (row) => row.description || '-'
  },
  {
    title: '修复建议',
    key: 'suggestion',
    minWidth: 280,
    ellipsis: {tooltip: NARROW_TOOLTIP_PROPS},
    render: (row) => row.suggestion || '-'
  }
]

function onRow(p: {row: Vuln}) {
  emit('row-click', p.row)
}

// 组件折叠面板的 key：保证唯一
function itemKey(c: {name: string; version: string}) {
  return `${c.name}::${c.version}`
}

// 严重度分布汇总（显示在面板 header 上）
function sevSummary(vulns: Vuln[]) {
  const c: Record<string, number> = {critical: 0, high: 0, medium: 0, low: 0, info: 0}
  for (const v of vulns) {
    const k = v.severity as string
    if (k in c) c[k]++
  }
  return c
}
</script>

<template>
  <div class="vuln-group-table">
    <!-- 顶部过滤区（始终显示，不参与折叠） -->
    <div class="filters">
      <NSpace align="center" :size="12" :wrap="false">
        <NInput
          v-model:value="search"
          placeholder="搜索 漏洞名称 / 编号 / 描述 / 建议"
          clearable
          style="width: 300px"
        />
        <NSelect
          v-model:value="severityFilter"
          multiple
          clearable
          placeholder="按风险等级筛选"
          :options="severityOptions"
          style="width: 220px"
        />
        <NText depth="3">
          共 {{ filteredComponents.length }} 个组件 / {{ totalVulnsShown }} 条漏洞
        </NText>
      </NSpace>
    </div>

    <NEmpty
      v-if="filteredComponents.length === 0"
      description="没有匹配的漏洞"
      style="margin-top: 40px"
    />

    <NCollapse
      v-else
      v-model:expanded-names="expandedNames"
      :trigger-areas="['main', 'arrow']"
      arrow-placement="right"
      class="group-collapse"
    >
      <NCollapseItem
        v-for="c in pagedComponents"
        :key="itemKey(c)"
        :name="itemKey(c)"
      >
        <!-- 面板 header：组件名 + 版本 + 许可证 + 语言 + 依赖方式 + 漏洞数 + 严重度分布 -->
        <template #header>
          <NSpace align="center" :size="10" :wrap="false" class="group-header">
            <NText strong class="group-name">{{ c.name }}</NText>
            <NText v-if="c.version" depth="3" code class="group-version">{{ c.version }}</NText>
            <!-- 许可证：v3.x 是字符串数组；为空或缺失显示 "未知" -->
            <NTooltip
              v-if="c.licenses && c.licenses.length > 0"
              placement="top"
              keep-alive-on-hover
            >
              <template #trigger>
                <NTag size="tiny" round type="info" class="group-license">
                  📜 {{ c.licenses[0] }}<template v-if="c.licenses.length > 1"> +{{ c.licenses.length - 1 }}</template>
                </NTag>
              </template>
              {{ c.licenses.join(', ') }}
            </NTooltip>
            <NTag v-else-if="c.licenses !== undefined" size="tiny" round class="group-license">📜 未知</NTag>
            <NTag v-if="c.language" size="tiny" round>{{ c.language }}</NTag>
            <!-- 依赖方式：true=直接依赖 / false=间接依赖 -->
            <NTag
              size="tiny"
              round
              :type="c.direct ? 'success' : 'default'"
              class="group-direct"
            >
              {{ c.direct ? '直接依赖' : '间接依赖' }}
            </NTag>
            <NText depth="3" class="group-count">{{ c.vulns.length }} 个漏洞</NText>
            <!-- 严重度分布 chip：按 critical → info 顺序 -->
            <NSpace :size="4" :wrap="false">
              <NTag
                v-if="sevSummary(c.vulns).critical > 0"
                size="tiny"
                type="error"
                round
              >{{ sevSummary(c.vulns).critical }} 严重</NTag>
              <NTag
                v-if="sevSummary(c.vulns).high > 0"
                size="tiny"
                type="warning"
                round
              >{{ sevSummary(c.vulns).high }} 高危</NTag>
              <NTag
                v-if="sevSummary(c.vulns).medium > 0"
                size="tiny"
                type="warning"
                round
              >{{ sevSummary(c.vulns).medium }} 中危</NTag>
              <NTag
                v-if="sevSummary(c.vulns).low > 0"
                size="tiny"
                type="info"
                round
              >{{ sevSummary(c.vulns).low }} 低危</NTag>
              <NTag
                v-if="sevSummary(c.vulns).info > 0"
                size="tiny"
                round
              >{{ sevSummary(c.vulns).info }} 提示</NTag>
            </NSpace>
          </NSpace>
        </template>

        <!-- 面板 body：该组件的漏洞表 -->
        <NDataTable
          :columns="columns"
          :data="c.vulns"
          :row-key="(row: Vuln) => row.id"
          :pagination="{pageSize: 10}"
          :bordered="false"
          striped
          size="small"
          :scroll-x="1400"
          style="margin-top: 4px"
          @row-click="onRow"
        />
      </NCollapseItem>
    </NCollapse>

    <!-- 外层分页器：按 10 个组件一页 -->
    <div v-if="filteredComponents.length > componentPageSize" class="outer-pager">
      <NPagination
        v-model:page="componentPage"
        :page-size="componentPageSize"
        :item-count="filteredComponents.length"
        show-size-picker
        :page-sizes="[5, 10, 20, 50]"
        size="small"
      />
    </div>
  </div>
</template>

<style scoped>
/* NTooltip 实际渲染到 body 末尾（teleport），普通 scoped 样式够不到。
   通过 contentClass="ellipsis-tooltip-popup" 给 .n-popover__content 本身加 hook，
   然后用 :global 选中它，约束宽度让长描述自动换行。
   160px ≈ 8 个中文字 / 16 个英文字（14px 字体下）。 */
:global(.ellipsis-tooltip-popup) {
  max-width: 160px !important;
  white-space: pre-wrap !important;
  word-break: break-word !important;
  line-height: 1.5 !important;
}

.vuln-group-table {
  display: flex;
  flex-direction: column;
  /* 与 VulnTable 同样的"固定高度 + 内部滚动"语义 */
  height: calc(100vh - 360px);
  min-height: 420px;
}
.filters {
  padding: 4px 0 12px;
  flex-shrink: 0;
}
.group-collapse {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
}
/* 让 NCollapse 内的 NDataTable 不要把面板高度撑爆 */
:deep(.n-collapse-item__content-inner) {
  padding-bottom: 8px;
}
.group-header {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex-wrap: nowrap;
}
.group-name {
  max-width: 360px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
/* 外层分页器：固定在折叠面板列表底部，不参与内部滚动 */
.outer-pager {
  flex-shrink: 0;
  display: flex;
  justify-content: center;
  padding: 12px 0 4px;
  border-top: 1px solid var(--n-border-color);
  background: var(--n-card-color);
}
.group-version {
  font-size: 12px;
}
.group-count {
  font-size: 12px;
}
</style>