<script setup lang="ts">
import {computed} from 'vue'
import {NGrid, NGi, NCard, NText, NSpace} from 'naive-ui'
import SeverityTag from './SeverityTag.vue'
import type {Report, Severity} from '@/api/types'

const props = defineProps<{report: Report | null}>()

const tiles = computed(() => {
  if (!props.report) {
    return [
      {label: '组件', value: 0, color: '#2080f0'},
      {label: '漏洞', value: 0, color: '#d03050'},
      {label: '严重', value: 0, color: '#d4380d'},
      {label: '高危', value: 0, color: '#fa541c'}
    ]
  }
  const c = props.report.severityCount || ({} as Record<Severity, number>)
  return [
    {label: '组件', value: props.report.totalComponents, color: '#2080f0'},
    {label: '漏洞', value: props.report.totalVulns, color: '#d03050'},
    {label: '严重', value: c.critical || 0, color: '#d4380d'},
    {label: '高危', value: c.high || 0, color: '#fa541c'},
    {label: '中危', value: c.medium || 0, color: '#faad14'},
    {label: '低危', value: c.low || 0, color: '#1890ff'}
  ]
})
</script>

<template>
  <NGrid :cols="6" :x-gap="12" responsive="screen" item-responsive>
    <NGi v-for="t in tiles" :key="t.label" :span="1">
      <NCard size="small" class="stat-tile">
        <NText depth="3" style="font-size: 12px">{{ t.label }}</NText>
        <div class="stat-value" :style="{color: t.color}">{{ t.value }}</div>
      </NCard>
    </NGi>
  </NGrid>
</template>

<style scoped>
.stat-tile {
  text-align: center;
}
.stat-value {
  font-size: 28px;
  font-weight: 600;
  margin-top: 4px;
  line-height: 1.2;
}
</style>