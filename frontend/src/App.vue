<script setup lang="ts">
import {computed, onMounted} from 'vue'
import {
  NConfigProvider,
  NMessageProvider,
  NDialogProvider,
  NLoadingBarProvider,
  NNotificationProvider,
  darkTheme,
  lightTheme,
  zhCN,
  dateZhCN
} from 'naive-ui'
import {useUIStore} from '@/stores/ui'
import {useConfigStore} from '@/stores/config'

const ui = useUIStore()
const cfg = useConfigStore()

const theme = computed(() => (ui.theme === 'dark' ? darkTheme : lightTheme))

onMounted(async () => {
  await cfg.load()
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
    :locale="zhCN"
    :date-locale="dateZhCN"
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