<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'
import {NGrid, NGi, NCard, NText, NSpace} from 'naive-ui'
import SeverityTag from './SeverityTag.vue'
import type {Report, Severity} from '@/api/types'

const props = defineProps<{report: Report | null}>()

const {t} = useI18n()

// computed：语言切换时 label 重新求值
const tiles = computed(() => {
  if (!props.report) {
    return [
      {label: t('stattiles.components'), value: 0, color: '#2080f0'},
      {label: t('stattiles.vulns'), value: 0, color: '#d03050'},
      {label: t('severity.critical'), value: 0, color: '#d4380d'},
      {label: t('severity.high'), value: 0, color: '#fa541c'}
    ]
  }
  const c = props.report.severityCount || ({} as Record<Severity, number>)
  return [
    {label: t('stattiles.components'), value: props.report.totalComponents, color: '#2080f0'},
    {label: t('stattiles.vulns'), value: props.report.totalVulns, color: '#d03050'},
    {label: t('severity.critical'), value: c.critical || 0, color: '#d4380d'},
    {label: t('severity.high'), value: c.high || 0, color: '#fa541c'},
    {label: t('severity.medium'), value: c.medium || 0, color: '#faad14'},
    {label: t('severity.low'), value: c.low || 0, color: '#1890ff'}
  ]
})
</script>

<template>
  <NGrid :cols="6" :x-gap="12" responsive="screen" item-responsive>
    <NGi v-for="tile in tiles" :key="tile.label" :span="1">
      <NCard size="small" class="stat-tile">
        <NText depth="3" style="font-size: 12px">{{ tile.label }}</NText>
        <div class="stat-value" :style="{color: tile.color}">{{ tile.value }}</div>
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
