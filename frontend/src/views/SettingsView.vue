<script setup lang="ts">
import {computed, h, nextTick, onBeforeUnmount, onMounted, ref} from 'vue'
import {useI18n} from 'vue-i18n'
import {
  NCard,
  NSpace,
  NInput,
  NInputNumber,
  NButton,
  NTag,
  NText,
  NSelect,
  NModal,
  NIcon,
  NMenu,
  NSkeleton,
  NDivider,
  NCollapse,
  NCollapseItem,
  NScrollbar,
  NCheckbox,
  useMessage,
  type MenuOption
} from 'naive-ui'
import {
  FolderOpenOutline,
  CheckmarkCircleOutline,
  CloseCircleOutline,
  CloudDownloadOutline,
  ReloadOutline,
  TerminalOutline,
  ShieldCheckmarkOutline,
  SettingsOutline,
  InformationCircleOutline,
  OptionsOutline
} from '@vicons/ionicons5'
import AppShell from '@/components/AppShell.vue'
import {useConfigStore} from '@/stores/config'
import {api} from '@/api'
import {applyLanguage} from '@/i18n'
import type {CliUpdateInfo} from '@/api/types'

const {t} = useI18n()
const cfg = useConfigStore()
const message = useMessage()

// 应用版本：由 vite define 注入（来源 wails.json 的 info.productVersion），模板里经此引用
const appVersion = __APP_VERSION__

const cliPath = ref(cfg.cliPath)
const verifying = ref(false)
const verifyResult = ref<{valid: boolean; version: string; message: string; rawOutput?: string} | null>(null)
const token = ref(cfg.token)
const verifyingToken = ref(false)
const tokenVerifyResult = ref<{valid: boolean; message: string; source: string} | null>(null)
const localDB = ref(cfg.localDB)
const concurrent = ref(cfg.maxConcurrent)
const theme = ref(cfg.theme)
const configPath = ref('')

// 界面语言：即时生效 + 立即持久化（不等"保存设置"，关掉应用也保留）
const language = ref<string>(cfg.language || 'zh-CN')
// 语言名用各自母语展示（通用约定：中文 / English），不随当前语言翻译
const languageOptions = [
  {label: '中文', value: 'zh-CN'},
  {label: 'English', value: 'en-US'}
]
function onLanguageChange(v: string) {
  language.value = v
  applyLanguage(v)
  void cfg.setLanguage(v).catch(() => {})
}

// 通用：报告位置
// useDefault 为 true 时走默认（文件夹扫描 = 项目本地，压缩包扫描 = AppData）
// useDefault 为 false 时使用 customPath（用户自定义输入框）
const folderReportUseDefault = ref(cfg.folderReportUseDefault)
const folderReportInput = ref<string>(
  cfg.folderReportUseDefault ? '' : (cfg.folderReportCustomPath || cfg.defaultReportsPath)
)
const zipReportUseDefault = ref(cfg.zipReportUseDefault)
const zipReportInput = ref<string>(
  cfg.zipReportUseDefault ? '' : (cfg.zipReportCustomPath || cfg.defaultReportsPath)
)

// 取消勾选时自动填默认 AppData 路径；勾选时清空（不展示但保留值）
function onFolderDefaultChange(v: boolean) {
  if (!v && !folderReportInput.value.trim()) {
    folderReportInput.value = cfg.defaultReportsPath || ''
  }
}
function onZipDefaultChange(v: boolean) {
  if (!v && !zipReportInput.value.trim()) {
    zipReportInput.value = cfg.defaultReportsPath || ''
  }
}

const themeOptions = computed(() => [
  {label: t('settings.runtime.themeLight'), value: 'light'},
  {label: t('settings.runtime.themeDark'), value: 'dark'}
])

// ---------- 左侧菜单（锚点跳转） ----------

type SectionKey = 'cli' | 'general' | 'vuln' | 'runtime' | 'about'
const activeSection = ref<SectionKey>('cli')

function renderIcon(icon: any) {
  return () => h(NIcon, null, () => h(icon))
}

// computed：语言切换时菜单项 label 重新求值
const sectionOptions = computed<MenuOption[]>(() => [
  {label: t('settings.nav.cli'), key: 'cli', icon: renderIcon(TerminalOutline)},
  {label: t('settings.nav.general'), key: 'general', icon: renderIcon(OptionsOutline)},
  {label: t('settings.nav.vuln'), key: 'vuln', icon: renderIcon(ShieldCheckmarkOutline)},
  {label: t('settings.nav.runtime'), key: 'runtime', icon: renderIcon(SettingsOutline)},
  {label: t('settings.nav.about'), key: 'about', icon: renderIcon(InformationCircleOutline)}
])

function pickSection(key: string) {
  activeSection.value = key as SectionKey
  const el = document.getElementById(`section-${key}`)
  if (el) {
    el.scrollIntoView({behavior: 'smooth', block: 'start'})
  }
}

// 滚动时自动高亮当前可见的 section。
let sectionObserver: IntersectionObserver | null = null
function setupSectionObserver() {
  sectionObserver = new IntersectionObserver(
    (entries) => {
      let bestEntry: IntersectionObserverEntry | null = null
      for (const e of entries) {
        if (!e.isIntersecting) continue
        if (!bestEntry || e.intersectionRatio > bestEntry.intersectionRatio) {
          bestEntry = e
        }
      }
      if (!bestEntry) return
      const id = (bestEntry.target as HTMLElement).id
      const key = id.replace(/^section-/, '') as SectionKey
      activeSection.value = key
    },
    {
      root: null,
      rootMargin: '-80px 0px -60% 0px',
      threshold: [0, 0.15, 0.3, 0.5, 0.75, 1]
    }
  )
  for (const k of ['cli', 'general', 'vuln', 'runtime', 'about'] as SectionKey[]) {
    const el = document.getElementById(`section-${k}`)
    if (el) sectionObserver.observe(el)
  }
}

onMounted(() => {
  nextTick(setupSectionObserver)
  loadConfigPath()
})

onBeforeUnmount(() => {
  sectionObserver?.disconnect()
  sectionObserver = null
})

// ---------- CLI ----------

async function pickCli() {
  try {
    const p = await api.PickExecutable()
    if (p) {
      cliPath.value = p
      verifyResult.value = null
    }
  } catch (e) {
    message.error(t('common.selectFailed', {msg: String(e)}))
  }
}

async function verify() {
  verifying.value = true
  try {
    const info = await cfg.setCliPath(cliPath.value.trim())
    verifyResult.value = {valid: info.valid, version: info.version, message: info.message, rawOutput: info.rawOutput ?? ''}
  } catch (e) {
    verifyResult.value = {valid: false, version: '', message: String(e), rawOutput: ''}
  } finally {
    verifying.value = false
  }
}

// ---------- Token 验证 ----------

async function verifyToken() {
  const tokenText = token.value.trim()
  if (!tokenText) {
    tokenVerifyResult.value = {valid: false, message: cfgTokenEmpty(), source: ''}
    message.warning(t('settings.vuln.tokenFillFirst'))
    return
  }
  verifyingToken.value = true
  tokenVerifyResult.value = null
  try {
    const info = await api.VerifyToken(tokenText)
    tokenVerifyResult.value = {valid: info.valid, message: info.message, source: info.source}
  } catch (e) {
    tokenVerifyResult.value = {valid: false, message: String(e), source: ''}
  } finally {
    verifyingToken.value = false
  }
}

// ---------- 更新 ----------

const updateModalVisible = ref(false)
const checking = ref(false)
const installing = ref(false)
const updateInfo = ref<CliUpdateInfo | null>(null)

const updateStatusTag = computed(() => {
  if (!updateInfo.value) return null
  if (updateInfo.value.hasUpdate) return {type: 'warning' as const, text: t('settings.updateModal.hasUpdate')}
  return {type: 'success' as const, text: t('settings.updateModal.isLatest')}
})

async function checkUpdate() {
  if (!cfg.cliPath) {
    message.warning(t('settings.updateModal.configCliFirst'))
    return
  }
  checking.value = true
  updateModalVisible.value = true
  updateInfo.value = null
  try {
    const info = await api.CheckCliUpdate(cfg.cliPath)
    updateInfo.value = info
  } catch (e) {
    message.error(t('common.checkUpdateFailed', {msg: String(e)}))
    updateInfo.value = {
      hasUpdate: false,
      currentVersion: cfg.cliVersion || '',
      latestVersion: '',
      releaseName: '',
      releaseURL: '',
      changelog: '',
      downloadURL: '',
      assetName: '',
      publishedAt: '',
      message: String(e)
    }
  } finally {
    checking.value = false
  }
}

async function installUpdate() {
  const info = updateInfo.value
  if (!info || !info.downloadURL) {
    message.warning(t('settings.updateModal.noDownloadAsset'))
    return
  }
  const target = cfg.cliPath || cliPath.value.trim()
  if (!target) {
    message.warning(t('settings.updateModal.missingTargetPath'))
    return
  }
  installing.value = true
  try {
    const res = await api.DownloadAndInstallCliUpdate(info.downloadURL, target)
    message.success(res.message + (res.backupPath ? t('settings.updateModal.backedUp', {path: res.backupPath}) : ''))
    if (cliPath.value !== cfg.cliPath) {
      cliPath.value = target
    }
    await verify()
    updateInfo.value = {...info, hasUpdate: false, message: t('settings.updateModal.updatedTo', {version: res.installedVersion})}
  } catch (e) {
    message.error(t('common.updateFailed', {msg: String(e)}))
  } finally {
    installing.value = false
  }
}

async function openReleasePage() {
  const url = updateInfo.value?.releaseURL
  if (!url) {
    message.warning(t('settings.updateModal.noReleaseLink'))
    return
  }
  try {
    await api.OpenReleasePage(url)
  } catch (e) {
    message.error(t('common.openFailed', {msg: String(e)}))
  }
}

function closeUpdateModal() {
  if (installing.value) return
  updateModalVisible.value = false
}

// ---------- 漏洞库 ----------

async function pickDB() {
  try {
    const p = await api.PickZip()
    if (p) localDB.value = p
  } catch (e) {
    message.error(t('common.selectFailed', {msg: String(e)}))
  }
}

// ---------- 关于（配置文件路径） ----------

async function loadConfigPath() {
  try {
    configPath.value = await api.GetConfigPath()
  } catch (e) {
    configPath.value = `(${t('settings.about.loadFailed')}: ${String(e)})`
  }
}

// ---------- 保存 ----------

async function saveAll() {
  try {
    if (cliPath.value !== cfg.cliPath) {
      await cfg.setCliPath(cliPath.value.trim())
      verifyResult.value = {valid: cfg.cliValid, version: cfg.cliVersion, message: ''}
    }
    if (token.value !== cfg.token) {
      await cfg.setToken(token.value)
    }
    if (localDB.value !== cfg.localDB) {
      await cfg.setLocalDB(localDB.value)
    }
    if (concurrent.value !== cfg.maxConcurrent) {
      await cfg.setMaxConcurrent(concurrent.value)
    }
    // 报告位置：useDefault=false 时把输入框的值作为 customPath 保存
    const folderCustom = folderReportInput.value.trim() || cfg.defaultReportsPath
    const zipCustom = zipReportInput.value.trim() || cfg.defaultReportsPath
    if (
      folderReportUseDefault.value !== cfg.folderReportUseDefault ||
      folderCustom !== cfg.folderReportCustomPath
    ) {
      await cfg.setFolderReportLocation(folderReportUseDefault.value, folderCustom)
    }
    if (
      zipReportUseDefault.value !== cfg.zipReportUseDefault ||
      zipCustom !== cfg.zipReportCustomPath
    ) {
      await cfg.setZipReportLocation(zipReportUseDefault.value, zipCustom)
    }
    message.success(t('common.saveSuccess'))
  } catch (e) {
    message.error(t('common.saveFailed', {msg: String(e)}))
  }
}

// 与 <script setup> 里遮蔽的 t 冲突，token 为空提示单独提取
function cfgTokenEmpty() {
  return t('settings.vuln.tokenEmpty')
}
</script>

<template>
  <AppShell>
    <div class="settings-layout">
      <!-- 左侧菜单：点击跳转锚点 -->
      <aside class="settings-nav">
        <NMenu
          :value="activeSection"
          :options="sectionOptions"
          :collapsed-width="64"
          :collapsed-icon-size="20"
          :indent="18"
          @update:value="pickSection"
        />
      </aside>

      <!-- 右侧内容：所有 section 始终显示，菜单点击只是平滑滚动 -->
      <main class="settings-content">
        <!-- CLI -->
        <NCard id="section-cli" :title="t('settings.card.cli')">
          <NSpace vertical :size="16">
            <div>
              <NText strong>{{ t('settings.cli.path') }}</NText>
              <NSpace :size="8" style="margin-top: 4px; flex-wrap: wrap">
                <NInput v-model:value="cliPath" placeholder="C:\\path\\to\\opensca-cli.exe" style="width: 420px" />
                <NButton @click="pickCli">{{ t('common.browse') }}</NButton>
                <NButton type="primary" :loading="verifying" @click="verify">{{ t('common.verify') }}</NButton>
                <NButton :loading="checking" @click="checkUpdate">
                  <template #icon>
                    <NIcon :component="CloudDownloadOutline" />
                  </template>
                  {{ t('common.update') }}
                </NButton>
              </NSpace>
              <div v-if="verifyResult" style="margin-top: 8px">
                <NSpace align="center">
                  <NTag v-if="verifyResult.valid" type="success" round>
                    {{ t('settings.cli.verifyPassed') }}
                  </NTag>
                  <NTag v-else type="error" round>
                    {{ t('settings.cli.verifyFailed') }}
                  </NTag>
                  <NText v-if="verifyResult.version" depth="3">{{ t('settings.cli.version', {v: verifyResult.version}) }}</NText>
                  <NText v-if="verifyResult.message" depth="3" style="font-size: 12px">{{ verifyResult.message }}</NText>
                </NSpace>
              </div>
            </div>
          </NSpace>
        </NCard>

        <!-- 通用 -->
        <NCard id="section-general" :title="t('settings.card.general')" style="margin-top: 16px">
          <NSpace vertical :size="20">
            <!-- 界面语言（置顶） -->
            <div>
              <NText strong>{{ t('settings.language.label') }}</NText>
              <div style="margin-top: 8px">
                <NSelect
                  :value="language"
                  :options="languageOptions"
                  style="width: 200px"
                  @update:value="onLanguageChange"
                />
              </div>
              <NText depth="3" style="font-size: 12px; display: block; margin-top: 4px">
                {{ t('settings.language.desc') }}
              </NText>
            </div>

            <NDivider />

            <!-- 文件夹扫描报告位置 -->
            <div>
              <NText strong>{{ t('settings.general.folderTitle') }}</NText>
              <div style="margin-top: 8px">
                <NCheckbox
                  v-model:checked="folderReportUseDefault"
                  @update:checked="onFolderDefaultChange"
                >
                  {{ t('settings.general.useDefault') }}
                </NCheckbox>
              </div>
              <NText depth="3" style="font-size: 12px; display: block; margin-top: 6px; line-height: 1.6">
                {{ t('settings.general.folderHint1') }} <code>.opensca-ui/reports</code>{{ t('settings.general.folderHint2') }}
                <code>.opensca-ui</code>{{ t('settings.general.folderHint3') }}
                <code>{{ cfg.defaultReportsPath || t('settings.general.defaultPathExample') }}</code>{{ t('settings.general.folderHint4') }}
              </NText>
              <div v-if="!folderReportUseDefault" style="margin-top: 10px">
                <NInput
                  v-model:value="folderReportInput"
                  :placeholder="t('settings.general.customPathPlaceholder')"
                  style="width: 520px"
                />
                <NText depth="3" style="font-size: 12px; display: block; margin-top: 4px">
                  {{ t('settings.general.autoSubdirs1') }} <code>reports/</code> {{ t('settings.general.autoSubdirs2') }} <code>logs/</code> {{ t('settings.general.autoSubdirs3') }}
                </NText>
              </div>
            </div>

            <NDivider />

            <!-- 压缩包扫描报告位置 -->
            <div>
              <NText strong>{{ t('settings.general.zipTitle') }}</NText>
              <div style="margin-top: 8px">
                <NCheckbox
                  v-model:checked="zipReportUseDefault"
                  @update:checked="onZipDefaultChange"
                >
                  {{ t('settings.general.useDefault') }}
                </NCheckbox>
              </div>
              <NText depth="3" style="font-size: 12px; display: block; margin-top: 6px; line-height: 1.6">
                {{ t('settings.general.zipHint') }} <code>{{ cfg.defaultReportsPath || t('settings.general.defaultPathExample') }}</code>{{ t('settings.general.folderHint4') }}
              </NText>
              <div v-if="!zipReportUseDefault" style="margin-top: 10px">
                <NInput
                  v-model:value="zipReportInput"
                  :placeholder="t('settings.general.customPathPlaceholder')"
                  style="width: 520px"
                />
                <NText depth="3" style="font-size: 12px; display: block; margin-top: 4px">
                  {{ t('settings.general.autoSubdirs1') }} <code>reports/</code> {{ t('settings.general.autoSubdirs2') }} <code>logs/</code> {{ t('settings.general.autoSubdirs3') }}
                </NText>
              </div>
            </div>
          </NSpace>
        </NCard>

        <!-- 漏洞库 -->
        <NCard id="section-vuln" :title="t('settings.card.vuln')" style="margin-top: 16px">
          <NSpace vertical :size="16">
            <div>
              <NText strong>{{ t('settings.vuln.token') }}</NText>
              <NSpace :size="8" align="center" style="margin-top: 4px">
                <NInput v-model:value="token" :placeholder="t('settings.vuln.tokenPlaceholder')" type="password" show-password-on="click" style="width: 420px" />
                <NButton :loading="verifyingToken" :disabled="!token.trim()" @click="verifyToken">
                  {{ t('common.verify') }}
                </NButton>
                <NTag v-if="tokenVerifyResult" :type="tokenVerifyResult.valid ? 'success' : 'error'" size="small">
                  {{ tokenVerifyResult.valid ? t('settings.vuln.valid') : t('settings.vuln.invalid') }}
                </NTag>
              </NSpace>
              <NText v-if="tokenVerifyResult" depth="3" style="font-size: 12px; display: block; margin-top: 4px">
                {{ tokenVerifyResult.message }}
                <span v-if="tokenVerifyResult.source"> · {{ t('settings.vuln.verifyEndpoint', {source: tokenVerifyResult.source}) }}</span>
              </NText>
              <NText depth="3" style="font-size: 12px">
                {{ t('settings.vuln.applyCloudBefore') }} <a href="https://opensca.xmirror.cn" target="_blank" rel="noopener" style="color: var(--n-primary-color)">opensca.xmirror.cn</a> {{ t('settings.vuln.applyCloudAfter') }}
              </NText>
              <NText depth="3" style="font-size: 12px; display: block; margin-top: 4px">
                {{ t('settings.vuln.tokenPassHint1') }} <code>-token</code> {{ t('settings.vuln.tokenPassHint2') }}
              </NText>
            </div>
            <div>
              <NText strong>{{ t('settings.vuln.localDB') }}</NText>
              <NSpace :size="8" style="margin-top: 4px">
                <NInput v-model:value="localDB" :placeholder="t('settings.vuln.localDBPlaceholder')" style="width: 420px" />
                <NButton @click="pickDB">{{ t('common.browse') }}</NButton>
              </NSpace>
              <NText depth="3" style="font-size: 12px; display: block; margin-top: 4px">
                {{ t('settings.vuln.localDBHint1') }} <code>-config</code> {{ t('settings.vuln.localDBHint2') }}
              </NText>
            </div>
          </NSpace>
        </NCard>

        <!-- 运行 -->
        <NCard id="section-runtime" :title="t('settings.card.runtime')" style="margin-top: 16px">
          <NSpace vertical :size="16">
            <div>
              <NText strong>{{ t('settings.runtime.maxConcurrent') }}</NText>
              <div style="margin-top: 4px">
                <NInputNumber v-model:value="concurrent" :min="1" :max="10" />
                <NText depth="3" style="margin-left: 12px; font-size: 12px">{{ t('settings.runtime.ioHint') }}</NText>
              </div>
            </div>
            <div>
              <NText strong>{{ t('settings.runtime.theme') }}</NText>
              <div style="margin-top: 4px">
                <NSelect v-model:value="theme" :options="themeOptions" style="width: 160px" />
              </div>
            </div>
          </NSpace>
        </NCard>

        <!-- 关于 -->
        <NCard id="section-about" :title="t('settings.card.about')" style="margin-top: 16px">
          <NSpace vertical :size="8">
            <NText>{{ t('settings.about.version', {v: appVersion}) }}</NText>
            <NText depth="3" style="font-size: 12px">
              {{ t('settings.about.builtWith') }}
            </NText>
            <NText depth="3" style="font-size: 12px">
              {{ t('settings.about.configPath') }}
              <code style="user-select: all">{{ configPath || t('settings.about.loading') }}</code>
            </NText>
          </NSpace>
        </NCard>

        <!-- 保存按钮 -->
        <div class="save-bar">
          <NSpace justify="end" align="center">
            <NButton type="primary" size="large" @click="saveAll">{{ t('settings.saveBar') }}</NButton>
          </NSpace>
        </div>
      </main>
    </div>

    <!-- 更新弹窗 -->
    <NModal
      v-model:show="updateModalVisible"
      preset="card"
      :title="t('settings.updateModal.title')"
      style="max-width: 640px"
      :mask-closable="!installing"
      :close-on-esc="!installing"
      @close="closeUpdateModal"
    >
      <NSkeleton v-if="checking" text :repeat="4" />

      <div v-else-if="updateInfo">
        <NSpace align="center" :size="12" style="margin-bottom: 12px">
          <NText>{{ t('settings.updateModal.current') }}</NText>
          <NTag round>v{{ updateInfo.currentVersion || t('settings.updateModal.unknown') }}</NTag>
          <NText>→</NText>
          <NText>{{ t('settings.updateModal.latest') }}</NText>
          <NTag :type="updateInfo.hasUpdate ? 'warning' : 'success'" round>
            v{{ updateInfo.latestVersion || t('settings.updateModal.unknown') }}
          </NTag>
          <NTag v-if="updateStatusTag" :type="updateStatusTag.type" size="small" round>
            {{ updateStatusTag.text }}
          </NTag>
        </NSpace>

        <NText v-if="updateInfo.message" depth="3" style="font-size: 12px">
          {{ updateInfo.message }}
        </NText>

        <NText v-if="updateInfo.assetName" depth="3" style="font-size: 12px; display: block; margin-top: 4px">
          {{ t('settings.updateModal.downloadAsset', {name: updateInfo.assetName}) }}
        </NText>

        <NCollapse v-if="updateInfo.changelog" style="margin-top: 12px">
          <NCollapseItem :title="t('settings.updateModal.changelog')" name="changelog">
            <NScrollbar style="max-height: 240px">
              <pre class="changelog">{{ updateInfo.changelog }}</pre>
            </NScrollbar>
          </NCollapseItem>
        </NCollapse>

        <NDivider />

        <NSpace justify="end">
          <NButton @click="closeUpdateModal" :disabled="installing">{{ t('settings.updateModal.close') }}</NButton>
          <NButton @click="openReleasePage" :disabled="installing">
            <template #icon>
              <NIcon :component="ReloadOutline" />
            </template>
            {{ t('settings.updateModal.openRelease') }}
          </NButton>
          <NButton
            v-if="updateInfo.hasUpdate && updateInfo.downloadURL"
            type="primary"
            :loading="installing"
            @click="installUpdate"
          >
            <template #icon>
              <NIcon :component="CloudDownloadOutline" />
            </template>
            {{ t('settings.updateModal.downloadReplace') }}
          </NButton>
        </NSpace>

        <NText depth="3" style="font-size: 12px; display: block; margin-top: 12px">
          {{ t('settings.updateModal.backupHint') }}
        </NText>
      </div>
    </NModal>
  </AppShell>
</template>

<style scoped>
.settings-layout {
  display: flex;
  gap: 24px;
  max-width: 960px;
  margin: 0 auto;
  align-items: flex-start;
}
.settings-nav {
  width: 180px;
  flex-shrink: 0;
  border-radius: 8px;
  background: var(--n-card-color);
  border: 1px solid var(--n-border-color);
  padding: 8px 0;
  position: sticky;
  top: 24px;
}
.settings-content {
  flex: 1;
  min-width: 0;
}
.save-bar {
  position: sticky;
  bottom: 16px;
  margin-top: 16px;
  padding: 12px 16px;
  background: var(--n-card-color);
  border: 1px solid var(--n-border-color);
  border-radius: 8px;
  box-shadow: 0 -4px 12px rgba(0, 0, 0, 0.04);
  z-index: 10;
}
.changelog {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: var(--n-font-family-mono);
  font-size: 12px;
  line-height: 1.6;
}
code {
  font-family: var(--n-font-family-mono);
  background: var(--n-action-color);
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 12px;
}
:deep(#section-cli),
:deep(#section-general),
:deep(#section-vuln),
:deep(#section-runtime),
:deep(#section-about) {
  scroll-margin-top: 80px;
}
</style>
