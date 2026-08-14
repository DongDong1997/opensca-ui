// 与 Go 后端对齐的类型定义。
//
// 不直接用 wailsjs 生成的 class（包含 convertValues 等反序列化辅助方法，污染类型约束）。
// 这里定义为 plain interface，与 wailsjs class 结构兼容（结构化类型），业务枚举用字面量类型。

export type TaskStatus = 'pending' | 'running' | 'success' | 'failed' | 'canceled'
export type Severity = 'critical' | 'high' | 'medium' | 'low' | 'info' | 'unknown'

export interface Task {
  id: string
  /** 任务备注（用户自由填）。当前会话内可空。 */
  label: string
  /** 项目名：取自文件夹名（目录）或压缩包去后缀，作为记录的绑定键。 */
  projectName: string
  path: string
  status: TaskStatus | string
  progress: number
  stage: string
  /** UnixMilli */
  startedAt: number
  /** UnixMilli */
  finishedAt: number
  durationMs: number
  exitCode: number
  error: string
  reportPath: string
  /** 可读 HTML 报告路径（opensca-cli 一次扫描同时输出） */
  htmlPath: string
  logPath: string
}

export interface TaskSummary {
  id: string
  label: string
  projectName: string
  path: string
  status: TaskStatus | string
  progress: number
  startedAt: number
  finishedAt: number
  durationMs: number
}

export interface StartScanRequest {
  path: string
  label?: string
}

export interface CliInfo {
  path: string
  valid: boolean
  version: string
  message: string
  /** CLI 的原始 stdout/stderr，便于诊断版本解析失败 */
  rawOutput?: string
}

export interface CliUpdateInfo {
  hasUpdate: boolean
  currentVersion: string
  latestVersion: string
  releaseName: string
  releaseURL: string
  changelog: string
  downloadURL: string
  assetName: string
  publishedAt: string
  message: string
}

export interface CliInstallResult {
  installedVersion: string
  backupPath: string
  targetPath: string
  message: string
}

/**
 * 最近打开的项目（首页展示用）。
 *
 * - path:        项目目录/压缩包的绝对路径
 * - label:       用户每次扫描的自由备注（可能为空，可能随扫描变化）
 * - projectName: 从 path 推导的稳定项目名（目录 basename / 压缩包去扩展名）。
 *                用作显示名 + 在 history 页做"按项目分组"的绑定键。
 *                与 label 的区别：label 会变，projectName 不会变。
 * - lastAt:      上次打开时间（UnixMilli）
 * - useCount:    累计扫描次数
 */
export interface RecentProject {
  path: string
  label: string
  projectName: string
  /** UnixMilli */
  lastAt: number
  useCount: number
}

/**
 * 项目本地的扫描记录（持久化在 <project>/.opensca-ui/scans.json，
 * 由 opensca-ui 在扫描完成时同步写入）。
 *
 * 与全局 history.json 字段一致，通过 taskID 关联。
 */
export interface ProjectScanEntry {
  id: string
  label: string
  projectName: string
  path: string
  status: string
  progress: number
  stage: string
  /** UnixMilli */
  startedAt: number
  /** UnixMilli */
  finishedAt: number
  durationMs: number
  exitCode: number
  error: string
  reportPath: string
  htmlPath: string
  logPath: string
}

export interface Config {
  cliPath: string
  cliValid: boolean
  cliVersion: string
  /** true = 用户手动配置过 CLI 路径；false = 启动时自动用内置 CLI（安装路径）作为默认 */
  cliPathManual: boolean
  maxConcurrent: number
  token: string
  localDB: string
  theme: 'light' | 'dark' | string
  /** 界面语言（zh-CN / en-US），与 Go Config.Language 对齐 */
  language: string
  /** 文件夹扫描：true = 项目本地 .opensca-ui/reports/（不可写回退 AppData） */
  folderReportUseDefault: boolean
  /** 文件夹扫描：useDefault=false 时使用该路径；空字符串回退 AppData */
  folderReportCustomPath: string
  /** 压缩包扫描：true = %APPDATA%/opensca-ui/reports/ */
  zipReportUseDefault: boolean
  /** 压缩包扫描：useDefault=false 时使用该路径；空字符串回退 AppData */
  zipReportCustomPath: string
}

export interface Vuln {
  id: string
  title: string
  severity: Severity | string
  cve: string[]
  cwe: string[]
  description: string
  suggestion: string
  references: string[]
  componentName: string
  componentVersion: string
  componentLanguage: string
  purl: string
  source: string
  /** v3.x 发布日期，例 "2023-10-24"（v2.x 缺省为空） */
  releaseDate: string
  /** v3.x 利用难度：1=易 2=中 3=难 0/缺省=未知 */
  exploitLevel: number
}

export interface Component {
  name: string
  version: string
  language: string
  purl: string
  vulns: Vuln[]
  /** v3.x 许可证列表（例 ["Apache-2.0", "MIT"]）；v2.x 或缺省为空数组 */
  licenses: string[]
  /** v3.x 依赖方式：true=直接依赖 false=间接依赖；v2.x 或缺省为 false */
  direct: boolean
}

export interface Report {
  taskId: string
  generatedAt: number
  totalComponents: number
  totalVulns: number
  severityCount: Record<string, number>
  components: Component[]
  /**
   * CLI 自身在 task_info.error 里给的状态消息（v3.x）。
   * 典型："not config vuln database origin" — 提示用户为什么没漏洞。
   * 没设值（v2.x 解析或一切正常）时为空串，UI 不展示。
   */
  warning: string
}

// 事件 payload：后端用 map[string]any 发射
export interface ScanLogEvent {
  taskID: string
  line: string
  ts: number
}

export interface ScanProgressEvent {
  taskID: string
  percent: number
  stage: string
}

export interface ScanStatusEvent {
  taskID: string
  status: string
}

export interface ScanDoneEvent {
  taskID: string
  status: string
  durationMs: number
  reportPath: string
}

export interface ScanErrorEvent {
  taskID?: string
  code: string
  message: string
}