<script setup lang="ts">
import {computed, h} from 'vue'
import {useRouter, useRoute, RouterLink} from 'vue-router'
import {useI18n} from 'vue-i18n'
import {
  NLayout,
  NLayoutHeader,
  NLayoutSider,
  NLayoutContent,
  NMenu,
  NIcon,
  NSwitch,
  NText,
  NSpace,
  NButton,
  NTag,
  type MenuOption
} from 'naive-ui'
import {
  ScanOutline,
  ListOutline,
  SettingsOutline,
  MoonOutline,
  SunnyOutline,
  ShieldCheckmarkOutline,
  HomeOutline,
  TimeOutline
} from '@vicons/ionicons5'
import {useUIStore} from '@/stores/ui'
import {useConfigStore} from '@/stores/config'
import logoUrl from '@/assets/logo.svg'

const ui = useUIStore()
const cfg = useConfigStore()
const router = useRouter()
const route = useRoute()
const {t} = useI18n()

function renderIcon(icon: any) {
  return () => h(NIcon, null, () => h(icon))
}

// computed：语言切换时 label 渲染函数重新求值
const menuOptions = computed<MenuOption[]>(() => [
  {label: () => h(RouterLink, {to: '/home'}, () => t('shell.nav.home')), key: 'home', icon: renderIcon(HomeOutline)},
  {label: () => h(RouterLink, {to: '/scan'}, () => t('shell.nav.scan')), key: 'scan', icon: renderIcon(ScanOutline)},
  {label: () => h(RouterLink, {to: '/tasks'}, () => t('shell.nav.tasks')), key: 'tasks', icon: renderIcon(ListOutline)},
  {label: () => h(RouterLink, {to: '/history'}, () => t('shell.nav.history')), key: 'history', icon: renderIcon(TimeOutline)},
  {label: () => h(RouterLink, {to: '/settings'}, () => t('shell.nav.settings')), key: 'settings', icon: renderIcon(SettingsOutline)}
])

const headerTitle = computed(() => {
  const key = route.meta.titleKey as string | undefined
  return key ? t(key) : 'OpenSCA UI'
})

// `tasks` / `tasks-history` / `tasks-running` / `tasks-finished` 都归到「任务管理」菜单项；
// 历史记录子页面顶栏照常显示"任务管理"，所以这里把 history 同族一并归类。
const activeKey = computed(() => {
  const n = route.name?.toString() ?? ''
  if (n === 'tasks' || n.startsWith('tasks-')) return 'tasks'
  return n || 'home'
})

const themeIcon = computed(() => (ui.theme === 'dark' ? SunnyOutline : MoonOutline))

function goSettings() {
  router.push('/settings')
}

// 顶栏右侧小提示：三态合并成一个统一的 chip
//   - 已查到更新信息：hasUpdate=true → 橙色"有更新 vX.X"；hasUpdate=false → 灰色"已是最新版"
//   - 没查到（CLI 未配置）：蓝色"未配置 CLI，点击前往设置"
//   - 查不到但已配置（网络失败）：不显示
const tipInfo = computed<{type: 'has' | 'latest' | 'unset'; text: string; title: string} | null>(() => {
  if (cfg.updateInfo) {
    if (cfg.updateInfo.hasUpdate) {
      return {
        type: 'has',
        text: t('shell.tip.hasUpdate', {version: cfg.updateInfo.latestVersion}),
        title: t('shell.tip.hasUpdateTitle', {version: cfg.updateInfo.latestVersion})
      }
    }
    return {
      type: 'latest',
      text: t('shell.tip.latest'),
      title: t('shell.tip.latestTitle')
    }
  }
  if (!cfg.cliPath) {
    return {
      type: 'unset',
      text: t('shell.tip.unset'),
      title: t('shell.tip.unsetTitle')
    }
  }
  return null
})
</script>

<template>
  <NLayout has-sider position="absolute" class="app-shell">
    <NLayoutSider
      bordered
      :collapsed="ui.sidebarCollapsed"
      collapse-mode="width"
      :collapsed-width="64"
      :width="220"
      show-trigger
      @collapse="ui.toggleSidebar"
      @expand="ui.toggleSidebar"
    >
      <div class="brand">
        <img :src="logoUrl" alt="logo" class="brand-logo" />
        <span v-if="!ui.sidebarCollapsed" class="brand-text">OpenSCA UI</span>
      </div>
      <NMenu
        :value="activeKey"
        :options="menuOptions"
        :collapsed="ui.sidebarCollapsed"
        :collapsed-width="64"
        :collapsed-icon-size="20"
      />
    </NLayoutSider>

    <NLayout>
      <NLayoutHeader bordered class="header">
        <NText strong style="font-size: 16px">{{ headerTitle }}</NText>
        <div class="header-right">
          <NSpace align="center">
            <!-- 顶栏始终显示 CLI 状态，方便确认版本 -->
            <NTag v-if="!cfg.cliPath" type="warning" size="small" round>{{ t('shell.cli.notConfigured') }}</NTag>
            <NTag v-else-if="cfg.cliValid" type="success" size="small" round>
              {{ t('shell.cli.valid', {version: cfg.cliVersion || 'unknown'}) }}
            </NTag>
            <NTag v-else type="error" size="small" round>
              {{ t('shell.cli.invalid', {reason: cfg.cliVersion || t('shell.cli.checkPath')}) }}
            </NTag>
            <!-- 更新状态小提示：启动时自动查一次，每次都显示当前状态，点击都跳设置 -->
            <div
              v-if="tipInfo"
              class="update-tip"
              :class="`update-tip--${tipInfo.type}`"
              role="button"
              tabindex="0"
              :title="tipInfo.title"
              @click="goSettings"
              @keydown.enter="goSettings"
            >
              {{ tipInfo.text }}
            </div>
            <NSwitch
              :value="ui.theme === 'dark'"
              size="medium"
              @update:value="(v) => ui.setTheme(v ? 'dark' : 'light')"
            >
              <template #icon>
                <NIcon :component="themeIcon" />
              </template>
            </NSwitch>
          </NSpace>
        </div>
      </NLayoutHeader>
      <NLayoutContent class="content" :native-scrollbar="false">
        <slot />
      </NLayoutContent>
    </NLayout>
  </NLayout>
</template>

<style scoped>
.app-shell {
  height: 100vh;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px;
  border-bottom: 1px solid var(--n-border-color);
  height: 56px;
  box-sizing: border-box;
}
.brand-logo {
  width: 24px;
  height: 24px;
  flex-shrink: 0;
  /* SVG 自带颜色；若用户换成 PNG 也保持 24px 尺寸 */
  object-fit: contain;
  display: block;
}
.brand-text {
  font-weight: 600;
  font-size: 16px;
}
.header {
  height: 56px;
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.header-right {
  display: flex;
  align-items: center;
}
.update-tip {
  display: inline-flex;
  align-items: center;
  padding: 2px 12px;
  margin-left: 4px;
  font-size: 12px;
  line-height: 20px;
  border-radius: 14px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s ease, transform 0.15s ease, border-color 0.15s ease, color 0.15s ease;
}
.update-tip:active {
  transform: scale(0.97);
}
.update-tip:focus-visible {
  outline: 2px solid currentColor;
  outline-offset: 2px;
}

/* 有新版本：橙色，更醒目 */
.update-tip--has {
  color: #d97706;
  border: 1px solid #f0a020;
  background: rgba(240, 160, 32, 0.08);
}
.update-tip--has:hover {
  background: rgba(240, 160, 32, 0.18);
}

/* 已是最新：灰色，柔和 */
.update-tip--latest {
  color: #909399;
  border: 1px solid #c0c4cc;
  background: rgba(144, 147, 153, 0.08);
}
.update-tip--latest:hover {
  background: rgba(144, 147, 153, 0.15);
}

/* 未配置：蓝色，引导用户去设置 */
.update-tip--unset {
  color: #2080f0;
  border: 1px solid #2080f0;
  background: rgba(32, 128, 240, 0.08);
}
.update-tip--unset:hover {
  background: rgba(32, 128, 240, 0.18);
}

/* 深色模式 */
:global(.dark) .update-tip--has,
:global([data-theme="dark"]) .update-tip--has {
  color: #f0a020;
  border-color: #f0a020;
  background: rgba(240, 160, 32, 0.12);
}
:global(.dark) .update-tip--latest,
:global([data-theme="dark"]) .update-tip--latest {
  color: #a6a9ad;
  border-color: #4a4d52;
  background: rgba(166, 169, 173, 0.06);
}
:global(.dark) .update-tip--unset,
:global([data-theme="dark"]) .update-tip--unset {
  color: #70c0ff;
  border-color: #2080f0;
  background: rgba(32, 128, 240, 0.12);
}
.content {
  padding: 24px;
  height: calc(100vh - 56px);
  overflow: auto;
}
</style>