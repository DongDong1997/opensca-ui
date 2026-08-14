<script setup lang="ts">
import {computed, nextTick, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {NEmpty, NScrollbar, NSwitch, NSpace, NText} from 'naive-ui'
import {useUIStore} from '@/stores/ui'
import type {LogEntry} from '@/stores/tasks'

const props = defineProps<{logs: LogEntry[]}>()
const ui = useUIStore()
const scrollRef = ref<InstanceType<typeof NScrollbar> | null>(null)
const {t} = useI18n()

watch(
  () => props.logs.length,
  async () => {
    if (!ui.logAutoScroll) return
    await nextTick()
    scrollRef.value?.scrollTo({top: 9999999, behavior: 'auto'})
  }
)

const text = computed(() => props.logs.map((l) => l.line).join('\n'))
</script>

<template>
  <div class="log-viewer">
    <div class="log-toolbar">
      <NSpace align="center" :size="8">
        <NText depth="3">{{ t('logviewer.autoScroll') }}</NText>
        <NSwitch :value="ui.logAutoScroll" @update:value="ui.setLogAutoScroll" size="small" />
        <NText depth="3" style="margin-left: 12px">{{ t('logviewer.lines', {n: logs.length}) }}</NText>
      </NSpace>
    </div>
    <NScrollbar ref="scrollRef" style="max-height: 480px">
      <NEmpty v-if="logs.length === 0" :description="t('logviewer.empty')" style="margin-top: 80px" />
      <pre v-else class="log-pre">{{ text }}</pre>
    </NScrollbar>
  </div>
</template>

<style scoped>
.log-viewer {
  border: 1px solid var(--n-border-color);
  border-radius: 6px;
  background: var(--n-card-color);
  overflow: hidden;
}
.log-toolbar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 6px 12px;
  border-bottom: 1px solid var(--n-border-color);
  background: var(--n-action-color);
}
.log-pre {
  margin: 0;
  padding: 12px 16px;
  font-family: 'Fira Code', 'Cascadia Code', Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--n-text-color);
}
</style>
