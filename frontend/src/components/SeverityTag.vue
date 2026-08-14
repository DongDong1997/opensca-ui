<script setup lang="ts">
import {computed} from 'vue'
import {NTag} from 'naive-ui'
import type {Severity} from '@/api/types'

const props = withDefaults(defineProps<{severity: Severity; size?: 'small' | 'medium' | 'large'; strong?: boolean}>(), {
  size: 'small',
  strong: true
})

const map: Record<Severity, {label: string; color: string; bg: string; border: string}> = {
  critical: {label: '严重', color: '#fff', bg: '#d4380d', border: '#d4380d'},
  high: {label: '高危', color: '#fff', bg: '#fa541c', border: '#fa541c'},
  medium: {label: '中危', color: '#fff', bg: '#faad14', border: '#faad14'},
  low: {label: '低危', color: '#fff', bg: '#1890ff', border: '#1890ff'},
  info: {label: '提示', color: '#fff', bg: '#909399', border: '#909399'},
  unknown: {label: '未知', color: '#fff', bg: '#909399', border: '#909399'}
}

const conf = computed(() => map[props.severity] ?? map.unknown)
</script>

<template>
  <NTag :size="size" round :color="{color: conf.bg, textColor: conf.color, borderColor: conf.border}" :strong="strong">
    {{ conf.label }}
  </NTag>
</template>