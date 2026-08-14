import {defineStore} from 'pinia'
import {computed, ref} from 'vue'
import {api} from '@/api'
import type {CliInfo, CliUpdateInfo, Config} from '@/api/types'

const DEFAULT_CONFIG: Config = {
  cliPath: '',
  cliValid: false,
  cliVersion: '',
  cliPathManual: false,
  maxConcurrent: 3,
  token: '',
  localDB: '',
  theme: 'light',
  folderReportUseDefault: true,
  folderReportCustomPath: '',
  zipReportUseDefault: true,
  zipReportCustomPath: ''
}

export const useConfigStore = defineStore('config', () => {
  const config = ref<Config>({...DEFAULT_CONFIG})
  const loaded = ref(false)
  const checking = ref(false)

  // 默认报告目录（%APPDATA%/opensca-ui/reports）。启动时拉一次。
  const defaultReportsPath = ref('')

  // CLI 更新信息（启动时自动查询，供顶栏小提示用）。
  const updateInfo = ref<CliUpdateInfo | null>(null)
  const checkingUpdate = ref(false)
  const lastUpdateCheckAt = ref(0)

  const cliPath = computed(() => config.value.cliPath)
  const cliValid = computed(() => config.value.cliValid)
  const cliVersion = computed(() => config.value.cliVersion)
  const maxConcurrent = computed(() => config.value.maxConcurrent)
  const token = computed(() => config.value.token)
  const localDB = computed(() => config.value.localDB)
  const theme = computed(() => config.value.theme)
  const folderReportUseDefault = computed(() => config.value.folderReportUseDefault)
  const folderReportCustomPath = computed(() => config.value.folderReportCustomPath)
  const zipReportUseDefault = computed(() => config.value.zipReportUseDefault)
  const zipReportCustomPath = computed(() => config.value.zipReportCustomPath)

  async function load() {
    try {
      const c = await api.GetConfig()
      config.value = {...DEFAULT_CONFIG, ...c}
    } catch (e) {
      // 后端未就绪时降级为默认值（开发期不会崩）
      console.warn('GetConfig failed:', e)
      config.value = {...DEFAULT_CONFIG}
    } finally {
      loaded.value = true
    }
    // 拉一次默认报告目录（用于取消勾选时填输入框）
    try {
      defaultReportsPath.value = await api.GetDefaultReportsPath()
    } catch {
      defaultReportsPath.value = ''
    }
  }

  async function checkCli(path?: string): Promise<CliInfo> {
    checking.value = true
    try {
      const target = path ?? config.value.cliPath
      // 注意：传了 path 时必须用 SetCliPath 而不是 CheckCli。
      // 后端 CheckCli 只刷新 Valid/Version，不动 CLIPath；
      // 如果在这里只调 CheckCli，路由放行后 StartScan 报"未配置"。
      const info = path !== undefined ? await api.SetCliPath(target) : await api.CheckCli(target)
      config.value.cliPath = info.path
      config.value.cliValid = info.valid
      config.value.cliVersion = info.version
      return info
    } finally {
      checking.value = false
    }
  }

  async function setCliPath(path: string): Promise<CliInfo> {
    await api.SetCliPath(path)
    config.value.cliPath = path
    const info = await checkCli(path)
    // 路径配好后顺便后台查一次更新
    void checkUpdate(true)
    return info
  }

  /**
   * 启动时/手动 触发一次后台更新检查。
   * 同一会话内 5 分钟最多查一次；无 cliPath / 已禁用 / 已经在查则跳过。
   * 静默失败，不弹错误（避免打扰用户）。
   */
  async function checkUpdate(force = false): Promise<void> {
    if (checkingUpdate.value) return
    if (!force && Date.now() - lastUpdateCheckAt.value < 5 * 60 * 1000) return
    if (!config.value.cliPath) {
      updateInfo.value = null
      return
    }
    checkingUpdate.value = true
    try {
      const info = await api.CheckCliUpdate(config.value.cliPath)
      updateInfo.value = info
    } catch (e) {
      // 网络失败静默
      console.warn('CheckCliUpdate failed:', e)
    } finally {
      checkingUpdate.value = false
      lastUpdateCheckAt.value = Date.now()
    }
  }

  async function setMaxConcurrent(n: number) {
    await api.SetMaxConcurrent(n)
    config.value.maxConcurrent = n
  }

  async function setToken(token: string) {
    await api.SetToken(token)
    config.value.token = token
  }

  async function setLocalDB(path: string) {
    await api.SetLocalDB(path)
    config.value.localDB = path
  }

  async function setFolderReportLocation(useDefault: boolean, customPath: string) {
    await api.SetFolderReportLocation(useDefault, customPath)
    config.value.folderReportUseDefault = useDefault
    config.value.folderReportCustomPath = customPath
  }

  async function setZipReportLocation(useDefault: boolean, customPath: string) {
    await api.SetZipReportLocation(useDefault, customPath)
    config.value.zipReportUseDefault = useDefault
    config.value.zipReportCustomPath = customPath
  }

  return {
    config,
    loaded,
    checking,
    updateInfo,
    checkingUpdate,
    defaultReportsPath,
    cliPath,
    cliValid,
    cliVersion,
    maxConcurrent,
    token,
    localDB,
    theme,
    folderReportUseDefault,
    folderReportCustomPath,
    zipReportUseDefault,
    zipReportCustomPath,
    load,
    checkCli,
    setCliPath,
    checkUpdate,
    setMaxConcurrent,
    setToken,
    setLocalDB,
    setFolderReportLocation,
    setZipReportLocation
  }
})