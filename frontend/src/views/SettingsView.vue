<script setup lang="ts">
import {computed, h, nextTick, onBeforeUnmount, onMounted, ref} from 'vue'
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
import type {CliUpdateInfo} from '@/api/types'

const cfg = useConfigStore()
const message = useMessage()

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

const themeOptions = [
  {label: '浅色', value: 'light'},
  {label: '深色', value: 'dark'}
]

// ---------- 左侧菜单（锚点跳转） ----------

type SectionKey = 'cli' | 'general' | 'vuln' | 'runtime' | 'about'
const activeSection = ref<SectionKey>('cli')

function renderIcon(icon: any) {
  return () => h(NIcon, null, () => h(icon))
}

const sectionOptions: MenuOption[] = [
  {label: 'CLI', key: 'cli', icon: renderIcon(TerminalOutline)},
  {label: '通用', key: 'general', icon: renderIcon(OptionsOutline)},
  {label: '漏洞库', key: 'vuln', icon: renderIcon(ShieldCheckmarkOutline)},
  {label: '运行', key: 'runtime', icon: renderIcon(SettingsOutline)},
  {label: '关于', key: 'about', icon: renderIcon(InformationCircleOutline)}
]

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
    message.error(`选择失败: ${String(e)}`)
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
  const t = token.value.trim()
  if (!t) {
    tokenVerifyResult.value = {valid: false, message: 'token 为空', source: ''}
    message.warning('请先填写 token')
    return
  }
  verifyingToken.value = true
  tokenVerifyResult.value = null
  try {
    const info = await api.VerifyToken(t)
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
  if (updateInfo.value.hasUpdate) return {type: 'warning' as const, text: '有新版本'}
  return {type: 'success' as const, text: '已是最新'}
})

async function checkUpdate() {
  if (!cfg.cliPath) {
    message.warning('请先配置 opensca-cli 路径')
    return
  }
  checking.value = true
  updateModalVisible.value = true
  updateInfo.value = null
  try {
    const info = await api.CheckCliUpdate(cfg.cliPath)
    updateInfo.value = info
  } catch (e) {
    message.error(`检查更新失败: ${String(e)}`)
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
    message.warning('没有可用的下载资产，请前往 release 页面手动下载')
    return
  }
  const target = cfg.cliPath || cliPath.value.trim()
  if (!target) {
    message.warning('缺少目标路径')
    return
  }
  installing.value = true
  try {
    const res = await api.DownloadAndInstallCliUpdate(info.downloadURL, target)
    message.success(res.message + (res.backupPath ? `（旧文件已备份到 ${res.backupPath}）` : ''))
    if (cliPath.value !== cfg.cliPath) {
      cliPath.value = target
    }
    await verify()
    updateInfo.value = {...info, hasUpdate: false, message: `已更新到 v${res.installedVersion}`}
  } catch (e) {
    message.error(`更新失败: ${String(e)}`)
  } finally {
    installing.value = false
  }
}

async function openReleasePage() {
  const url = updateInfo.value?.releaseURL
  if (!url) {
    message.warning('没有 release 链接')
    return
  }
  try {
    await api.OpenReleasePage(url)
  } catch (e) {
    message.error(`打开失败: ${String(e)}`)
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
    message.error(`选择失败: ${String(e)}`)
  }
}

// ---------- 关于（配置文件路径） ----------

async function loadConfigPath() {
  try {
    configPath.value = await api.GetConfigPath()
  } catch (e) {
    configPath.value = '(获取失败: ' + String(e) + ')'
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
    message.success('设置已保存')
  } catch (e) {
    message.error(`保存失败: ${String(e)}`)
  }
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
        <NCard id="section-cli" title="CLI 设置">
          <NSpace vertical :size="16">
            <div>
              <NText strong>opensca-cli 路径</NText>
              <NSpace :size="8" style="margin-top: 4px; flex-wrap: wrap">
                <NInput v-model:value="cliPath" placeholder="C:\\path\\to\\opensca-cli.exe" style="width: 420px" />
                <NButton @click="pickCli">浏览</NButton>
                <NButton type="primary" :loading="verifying" @click="verify">验证</NButton>
                <NButton :loading="checking" @click="checkUpdate">
                  <template #icon>
                    <NIcon :component="CloudDownloadOutline" />
                  </template>
                  更新
                </NButton>
              </NSpace>
              <div v-if="verifyResult" style="margin-top: 8px">
                <NSpace align="center">
                  <NTag v-if="verifyResult.valid" type="success" round>
                    ✓ 验证通过
                  </NTag>
                  <NTag v-else type="error" round>
                    ✗ 验证失败
                  </NTag>
                  <NText v-if="verifyResult.version" depth="3">版本: {{ verifyResult.version }}</NText>
                  <NText v-if="verifyResult.message" depth="3" style="font-size: 12px">{{ verifyResult.message }}</NText>
                </NSpace>
              </div>
            </div>
          </NSpace>
        </NCard>

        <!-- 通用 -->
        <NCard id="section-general" title="通用" style="margin-top: 16px">
          <NSpace vertical :size="20">
            <!-- 文件夹扫描报告位置 -->
            <div>
              <NText strong>生成报告位置（文件夹扫描）</NText>
              <div style="margin-top: 8px">
                <NCheckbox
                  v-model:checked="folderReportUseDefault"
                  @update:checked="onFolderDefaultChange"
                >
                  是否使用默认位置
                </NCheckbox>
              </div>
              <NText depth="3" style="font-size: 12px; display: block; margin-top: 6px; line-height: 1.6">
                选择文件夹扫描时，默认生成在文件夹根目录的 <code>.opensca-ui/reports</code> 中。
                无法生成 <code>.opensca-ui</code> 时，默认位置在
                <code>{{ cfg.defaultReportsPath || 'C:\\Users\\&lt;用户名&gt;\\AppData\\Roaming\\opensca-ui\\reports' }}</code>。
              </NText>
              <div v-if="!folderReportUseDefault" style="margin-top: 10px">
                <NInput
                  v-model:value="folderReportInput"
                  placeholder="自定义报告目录路径"
                  style="width: 520px"
                />
                <NText depth="3" style="font-size: 12px; display: block; margin-top: 4px">
                  路径下会自动创建 <code>reports/</code> 与 <code>logs/</code> 子目录。
                </NText>
              </div>
            </div>

            <NDivider />

            <!-- 压缩包扫描报告位置 -->
            <div>
              <NText strong>生成报告位置（压缩包扫描）</NText>
              <div style="margin-top: 8px">
                <NCheckbox
                  v-model:checked="zipReportUseDefault"
                  @update:checked="onZipDefaultChange"
                >
                  是否使用默认位置
                </NCheckbox>
              </div>
              <NText depth="3" style="font-size: 12px; display: block; margin-top: 6px; line-height: 1.6">
                默认位置就是 <code>{{ cfg.defaultReportsPath || 'C:\\Users\\&lt;用户名&gt;\\AppData\\Roaming\\opensca-ui\\reports' }}</code>。
              </NText>
              <div v-if="!zipReportUseDefault" style="margin-top: 10px">
                <NInput
                  v-model:value="zipReportInput"
                  placeholder="自定义报告目录路径"
                  style="width: 520px"
                />
                <NText depth="3" style="font-size: 12px; display: block; margin-top: 4px">
                  路径下会自动创建 <code>reports/</code> 与 <code>logs/</code> 子目录。
                </NText>
              </div>
            </div>
          </NSpace>
        </NCard>

        <!-- 漏洞库 -->
        <NCard id="section-vuln" title="漏洞库" style="margin-top: 16px">
          <NSpace vertical :size="16">
            <div>
              <NText strong>云漏洞库 Token</NText>
              <NSpace :size="8" align="center" style="margin-top: 4px">
                <NInput v-model:value="token" placeholder="可选，留空时仅使用本地漏洞库" type="password" show-password-on="click" style="width: 420px" />
                <NButton :loading="verifyingToken" :disabled="!token.trim()" @click="verifyToken">
                  验证
                </NButton>
                <NTag v-if="tokenVerifyResult" :type="tokenVerifyResult.valid ? 'success' : 'error'" size="small">
                  {{ tokenVerifyResult.valid ? '✓ 有效' : '✗ 无效' }}
                </NTag>
              </NSpace>
              <NText v-if="tokenVerifyResult" depth="3" style="font-size: 12px; display: block; margin-top: 4px">
                {{ tokenVerifyResult.message }}
                <span v-if="tokenVerifyResult.source"> · 验证端点: {{ tokenVerifyResult.source }}</span>
              </NText>
              <NText depth="3" style="font-size: 12px">
                去 <a href="https://opensca.xmirror.cn" target="_blank" rel="noopener" style="color: var(--n-primary-color)">opensca.xmirror.cn</a> 免费申请
              </NText>
              <NText depth="3" style="font-size: 12px; display: block; margin-top: 4px">
                opensca-ui 会作为 <code>-token</code> 参数传给 CLI（v2.x/v3.x 均支持），无需手动配 config.json。
              </NText>
            </div>
            <div>
              <NText strong>本地漏洞库（db.json）</NText>
              <NSpace :size="8" style="margin-top: 4px">
                <NInput v-model:value="localDB" placeholder="可选，db.json 完整路径" style="width: 420px" />
                <NButton @click="pickDB">浏览</NButton>
              </NSpace>
              <NText depth="3" style="font-size: 12px; display: block; margin-top: 4px">
                v3.x 通过 <code>-config</code> 注入；opensca-ui 会自动生成临时 config.json（只填 <code>origin.json</code>）传给 CLI。
              </NText>
            </div>
          </NSpace>
        </NCard>

        <!-- 运行 -->
        <NCard id="section-runtime" title="运行设置" style="margin-top: 16px">
          <NSpace vertical :size="16">
            <div>
              <NText strong>最大并发扫描数</NText>
              <div style="margin-top: 4px">
                <NInputNumber v-model:value="concurrent" :min="1" :max="10" />
                <NText depth="3" style="margin-left: 12px; font-size: 12px">建议 2-4，过高会竞争 IO</NText>
              </div>
            </div>
            <div>
              <NText strong>界面主题</NText>
              <div style="margin-top: 4px">
                <NSelect v-model:value="theme" :options="themeOptions" style="width: 160px" />
              </div>
            </div>
          </NSpace>
        </NCard>

        <!-- 关于 -->
        <NCard id="section-about" title="关于" style="margin-top: 16px">
          <NSpace vertical :size="8">
            <NText>OpenSCA UI v0.1.0</NText>
            <NText depth="3" style="font-size: 12px">
              本软件是 opensca-cli 的桌面图形界面，基于 Wails + Vue 3 + Naive UI 构建。
            </NText>
            <NText depth="3" style="font-size: 12px">
              配置文件位置：
              <code style="user-select: all">{{ configPath || '加载中…' }}</code>
            </NText>
          </NSpace>
        </NCard>

        <!-- 保存按钮 -->
        <div class="save-bar">
          <NSpace justify="end" align="center">
            <NButton type="primary" size="large" @click="saveAll">保存设置</NButton>
          </NSpace>
        </div>
      </main>
    </div>

    <!-- 更新弹窗 -->
    <NModal
      v-model:show="updateModalVisible"
      preset="card"
      title="检查 opensca-cli 更新"
      style="max-width: 640px"
      :mask-closable="!installing"
      :close-on-esc="!installing"
      @close="closeUpdateModal"
    >
      <NSkeleton v-if="checking" text :repeat="4" />

      <div v-else-if="updateInfo">
        <NSpace align="center" :size="12" style="margin-bottom: 12px">
          <NText>当前</NText>
          <NTag round>v{{ updateInfo.currentVersion || '未知' }}</NTag>
          <NText>→</NText>
          <NText>最新</NText>
          <NTag :type="updateInfo.hasUpdate ? 'warning' : 'success'" round>
            v{{ updateInfo.latestVersion || '未知' }}
          </NTag>
          <NTag v-if="updateStatusTag" :type="updateStatusTag.type" size="small" round>
            {{ updateStatusTag.text }}
          </NTag>
        </NSpace>

        <NText v-if="updateInfo.message" depth="3" style="font-size: 12px">
          {{ updateInfo.message }}
        </NText>

        <NText v-if="updateInfo.assetName" depth="3" style="font-size: 12px; display: block; margin-top: 4px">
          下载资产：{{ updateInfo.assetName }}
        </NText>

        <NCollapse v-if="updateInfo.changelog" style="margin-top: 12px">
          <NCollapseItem title="更新日志" name="changelog">
            <NScrollbar style="max-height: 240px">
              <pre class="changelog">{{ updateInfo.changelog }}</pre>
            </NScrollbar>
          </NCollapseItem>
        </NCollapse>

        <NDivider />

        <NSpace justify="end">
          <NButton @click="closeUpdateModal" :disabled="installing">关闭</NButton>
          <NButton @click="openReleasePage" :disabled="installing">
            <template #icon>
              <NIcon :component="ReloadOutline" />
            </template>
            打开 release 页面
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
            下载并替换
          </NButton>
        </NSpace>

        <NText depth="3" style="font-size: 12px; display: block; margin-top: 12px">
          提示：替换前会自动备份原文件为 <code>opensca-cli.exe.bak</code>，失败可手动恢复。
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