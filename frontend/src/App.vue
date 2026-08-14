<script setup lang="ts">
import {computed, onMounted, watch} from 'vue'
import {useRoute} from 'vue-router'
import {
  NConfigProvider,
  NMessageProvider,
  NDialogProvider,
  NLoadingBarProvider,
  NNotificationProvider,
  darkTheme,
  lightTheme,
  zhCN,
  dateZhCN,
  enUS,
  dateEnUS
} from 'naive-ui'
import {useUIStore} from '@/stores/ui'
import {useConfigStore} from '@/stores/config'
import {i18n, applyLanguage} from '@/i18n'

const ui = useUIStore()
const cfg = useConfigStore()
const route = useRoute()

const theme = computed(() => (ui.theme === 'dark' ? darkTheme : lightTheme))

// Naive UI 的组件文案随界面语言切换（:locale 与 :date-locale 都要换）
const naiveLocale = computed(() => (i18n.global.locale.value === 'en-US' ? enUS : zhCN))
const naiveDateLocale = computed(() => (i18n.global.locale.value === 'en-US' ? dateEnUS : dateZhCN))

// 浏览器标题跟随路由与语言（guard 只做一次性跳转，这里响应式覆盖语言切换）
function setDocumentTitle() {
  const key = route.meta.titleKey as string | undefined
  document.title = key ? `${i18n.global.t(key)} · OpenSCA UI` : 'OpenSCA UI'
}
watch(() => [route.meta.titleKey, i18n.global.locale.value], setDocumentTitle, {immediate: true})

onMounted(async () => {
  await cfg.load()
  // 应用持久化的语言（guard 已设过一次，这里双保险覆盖绕过 guard 的路径）
  applyLanguage(cfg.language || 'zh-CN')
  // 应用持久化的主题
  ui.setTheme((cfg.theme as 'light' | 'dark') || 'light')
  // 启动后后台检查 CLI 更新（仅当已经配置过路径才查），不阻塞 UI
  if (cfg.cliPath) {
    void cfg.checkUpdate(true)
  }
})
</script>

<template>
  <NConfigProvider
    :theme="theme"
    :theme-overrides="{}"
    :locale="naiveLocale"
    :date-locale="naiveDateLocale"
    preflight-style-disabled
  >
    <NLoadingBarProvider>
      <NMessageProvider>
        <NNotificationProvider>
          <NDialogProvider>
            <RouterView />
          </NDialogProvider>
        </NNotificationProvider>
      </NMessageProvider>
    </NLoadingBarProvider>
  </NConfigProvider>
</template>

<style>
html, body, #app {
  height: 100%;
  margin: 0;
  padding: 0;
}
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
    'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
</style>