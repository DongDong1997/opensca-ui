<script setup lang="ts">
import {computed, h, ref} from 'vue'
import {useI18n} from 'vue-i18n'
import {NDataTable, NTag, NText, NEmpty, NSelect, NSpace, NInput, type DataTableColumns, type DataTableRowKey} from 'naive-ui'
import SeverityTag from './SeverityTag.vue'
import type {Vuln, Severity} from '@/api/types'

const props = defineProps<{vulns: Vuln[]}>()
const emit = defineEmits<{(e: 'row-click', v: Vuln): void}>()

const {t} = useI18n()

const search = ref('')
const severityFilter = ref<Severity[]>([])

// computed：语言切换时选项 label 重新求值
const severityOptions = computed(() => [
  {label: t('severity.critical'), value: 'critical'},
  {label: t('severity.high'), value: 'high'},
  {label: t('severity.medium'), value: 'medium'},
  {label: t('severity.low'), value: 'low'},
  {label: t('severity.info'), value: 'info'}
])

const filtered = computed(() => {
  let arr = props.vulns
  if (severityFilter.value.length > 0) {
    arr = arr.filter((v) => severityFilter.value.includes(v.severity as Severity))
  }
  if (search.value.trim()) {
    const q = search.value.trim().toLowerCase()
    arr = arr.filter(
      (v) =>
        v.title?.toLowerCase().includes(q) ||
        v.id?.toLowerCase().includes(q) ||
        v.cve?.some((c) => c.toLowerCase().includes(q)) ||
        v.componentName?.toLowerCase().includes(q)
    )
  }
  return arr
})

// computed：locale 变化时列头重渲染（title 直接调用 t()，保证 computed 依赖 locale）
const columns = computed<DataTableColumns<Vuln>>(() => [
  {
    title: t('vuln.severityCol'),
    key: 'severity',
    width: 90,
    render: (row) => h(SeverityTag, {severity: row.severity as Severity})
  },
  {
    title: t('vuln.vulnId'),
    key: 'id',
    width: 160,
    render: (row) => h(NText, {code: true, depth: 3}, () => row.id || '-')
  },
  {
    title: t('vuln.cve'),
    key: 'cve',
    width: 160,
    render: (row) => {
      if (!row.cve?.length) return '-'
      return h(
        NSpace,
        {size: 4},
        () => row.cve.map((c) => h(NTag, {size: 'tiny', round: true}, () => c))
      )
    }
  },
  {
    title: t('vuln.title'),
    key: 'title',
    ellipsis: {tooltip: true},
    render: (row) => row.title || row.description?.slice(0, 60) || '-'
  },
  {
    title: t('vuln.affectedComponent'),
    key: 'component',
    width: 240,
    render: (row) =>
      h('div', {style: 'display:flex; flex-direction:column;'}, [
        h(NText, {}, () => row.componentName || '-'),
        h(NText, {depth: 3, code: true, style: 'font-size: 12px'}, () => row.componentVersion || '')
      ])
  },
  {
    title: t('vuln.solution'),
    key: 'suggestion',
    width: 280,
    ellipsis: {tooltip: true},
    render: (row) => row.suggestion || '-'
  }
])

function onRow(props: {row: Vuln}) {
  emit('row-click', props.row)
}

const checkedRowKeys = ref<DataTableRowKey[]>([])
</script>

<template>
  <div class="vuln-table">
    <div class="filters">
      <NSpace align="center" :size="12">
        <NInput
          v-model:value="search"
          :placeholder="t('vuln.searchSimplePlaceholder')"
          clearable
          style="width: 280px"
        />
        <NSelect
          v-model:value="severityFilter"
          multiple
          clearable
          :placeholder="t('vuln.severityFilter')"
          :options="severityOptions"
          style="width: 220px"
        />
        <NText depth="3">{{ t('vuln.matchCount', {n: filtered.length}) }}</NText>
      </NSpace>
    </div>
    <NDataTable
      :columns="columns"
      :data="filtered"
      :row-key="(row: Vuln) => row.id + ':' + row.componentName + ':' + row.componentVersion"
      :row-class-name="() => 'clickable-row'"
      :pagination="{pageSize: 20}"
      :bordered="false"
      striped
      class="vuln-table-body"
      @row-click="onRow"
    />
    <NEmpty v-if="filtered.length === 0" :description="t('vuln.noMatch')" style="margin-top: 40px" />
  </div>
</template>

<style scoped>
.vuln-table {
  display: flex;
  flex-direction: column;
  /* 固定高度：父级 AppShell.content = 100vh - 56px，外层 NSpace padding 24*2=48、
     NCard padding ~24，再加上顶栏/StatTiles/NAlert 累计 ~270px。剩余给表格。
     留出 360px 给 chrome，min-height 保证小屏也能撑开滚动区。 */
  height: calc(100vh - 360px);
  min-height: 420px;
}
.filters {
  padding: 4px 0;
  flex-shrink: 0;
}
.vuln-table-body {
  flex: 1 1 auto;
  min-height: 0;
  margin-top: 12px;
}
/* 让 NDataTable 内部 wrapper 滚动（head 固定、分页器固定） */
:deep(.vuln-table-body .n-data-table) {
  height: 100%;
}
:deep(.vuln-table-body .n-data-table-wrapper) {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
}
:deep(.vuln-table-body .n-data-table__pagination) {
  flex-shrink: 0;
}
:deep(.clickable-row) {
  cursor: pointer;
}
</style>
