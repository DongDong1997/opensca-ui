<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'
import {NTag} from 'naive-ui'
import type {Severity} from '@/api/types'

const props = withDefaults(defineProps<{severity: Severity; size?: 'small' | 'medium' | 'large'; strong?: boolean}>(), {
  size: 'small',
  strong: true
})

const {t} = useI18n()

// 颜色静态，label 随语言取 t()（computed 保证 locale 变化时重新求值）
const map: Record<Severity, {labelKey: string; color: string; bg: string; border: string}> = {
  critical: {labelKey: 'severity.critical', color: '#fff', bg: '#d4380d', border: '#d4380d'},
  high: {labelKey: 'severity.high', color: '#fff', bg: '#fa541c', border: '#fa541c'},
  medium: {labelKey: 'severity.medium', color: '#fff', bg: '#faad14', border: '#faad14'},
  low: {labelKey: 'severity.low', color: '#fff', bg: '#1890ff', border: '#1890ff'},
  info: {labelKey: 'severity.info', color: '#fff', bg: '#909399', border: '#909399'},
  unknown: {labelKey: 'severity.unknown', color: '#fff', bg: '#909399', border: '#909399'}
}

const conf = computed(() => {
  const base = map[props.severity] ?? map.unknown
  return {...base, label: t(base.labelKey)}
})
</script>

<template>
  <NTag :size="size" round :color="{color: conf.bg, textColor: conf.color, borderColor: conf.border}" :strong="strong">
    {{ conf.label }}
  </NTag>
</template>
