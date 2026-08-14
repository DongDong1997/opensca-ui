# OpenSCA UI

桌面图形界面版 [opensca-cli](https://github.com/XmirrorSecurity/OpenSCA-cli)：基于 **Wails v2 + Vue 3 + Naive UI**，单 exe 文件（~12MB），双击即开浏览器。

## 功能

- 📁 选择项目目录 / 压缩包 / 拖拽上传
- 🚀 多任务并发扫描（默认 3，可调）
- 📜 实时日志流 + 进度条
- 🔍 漏洞结果可视化（表格 + 严重度筛选 + 详情抽屉）
- 💾 报告导出（HTML / JSON / Excel，由 opensca-cli 原生生成）
- 🌗 浅色 / 深色模式
- ⚙️ CLI 路径、token、本地漏洞库、并发上限配置

## 快速开始

### 开发模式

```bash
cd opensca-ui
wails dev          # 自动启动 Vite dev server + Go 后端
```

首次运行会自动生成 `frontend/wailsjs/` 绑定。

### 生产构建

```bash
powershell -ExecutionPolicy Bypass -File scripts\fetch-cli.ps1   # ① 拉取内置 opensca-cli（如已放置可跳过）
wails build               # 单文件 exe 到 build/bin/opensca-ui.exe（已内嵌 opensca-cli）
wails build -nsis         # 加 NSIS 安装包（需先安装 NSIS）
```

### 内置 opensca-cli

应用把 **opensca-cli.exe 直接打包进安装包**（`go:embed`），用户装完软件即可扫描，
**无需再单独下载 opensca-cli**。NSIS 安装器会把内置 CLI 装到**安装路径**
（与应用 exe 同一目录），并作为默认 CLI 路径；用户仍可在设置页改成别的路径（设置都会保留）。

- 内置文件位置：`internal\bundle\opensca-cli.exe`（真实 exe，约 20MB）
- 安装后位置：`安装路径\opensca-cli.exe`（与 `opensca-ui.exe` 同目录）
- 更新内置版本：运行 `scripts\fetch-cli.ps1`（拉最新版）或手动覆盖该文件
- 构建发布包前请确认该文件是真实 exe（build 脚本会校验，占位文件会告警）
- 单文件 exe / 直接拷贝运行：首次启动会从嵌入字节把 CLI 解包到应用同目录
- 用户已配置过路径时，启动不会覆盖其设置；安装路径下已被"下载并替换"
  更新过的 CLI 也不会被内置旧版覆盖

## 项目结构

```
opensca-ui/
├── main.go              # Wails 启动入口
├── app.go               # 绑定到前端的方法（GetConfig/StartScan/CancelScan/...）
├── wails.json           # Wails 构建配置
├── internal/
│   ├── bundle/          # 内置 opensca-cli（go:embed 打包，安装到应用同目录）
│   ├── config/          # 配置管理（%APPDATA%/opensca-ui/config.json）
│   ├── scanner/         # 扫描任务管理器（worker pool + subprocess）
│   └── platform/        # 跨平台路径（%APPDATA% 等）
├── scripts/
│   └── fetch-cli.ps1    # 拉取最新 opensca-cli 放入 internal/bundle/
├── frontend/
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── wailsjs/         # Wails 自动生成的 JS/TS 绑定（勿手改）
│   └── src/
│       ├── main.ts      # Vue 入口
│       ├── App.vue      # 顶层 Providers（Naive UI + Router）
│       ├── router.ts    # vue-router 配置（含未配置 CLI 时强制跳 /welcome）
│       ├── api/
│       │   ├── index.ts   # 绑定集中重导出
│       │   └── types.ts   # 业务类型（Task/Report/Vuln/...）
│       ├── composables/   # useWailsEvent/useTaskStream/useTheme
│       ├── stores/        # Pinia: tasks/config/ui
│       ├── components/    # DropZone/TaskCard/LogViewer/VulnTable/...
│       └── views/         # Welcome/Scan/Tasks/Report/Settings
└── build/               # Wails 资源 + 构建产物
    └── bin/
        └── opensca-ui.exe
```

## 技术栈

- **后端**：Go 1.23+ / Wails v2.14 / 子进程调用 opensca-cli
- **前端**：Vue 3.5 / TypeScript 5.6 / Vite 7
- **UI 库**：Naive UI 2.40（暗色主题支持）
- **状态管理**：Pinia 2.2
- **路由**：vue-router 4（hash mode，桌面应用无服务端）

## 数据落盘位置

| 内容 | 路径 |
|---|---|
| 配置 | `%APPDATA%/opensca-ui/config.json` |
| 内置 CLI | `安装路径/opensca-cli.exe`（与应用 exe 同目录） |
| 任务报告 | `%APPDATA%/opensca-ui/reports/<taskID>.json` |
| 任务日志 | `%APPDATA%/opensca-ui/logs/<taskID>.log` |

## 事件协议（Go → 前端）

| 事件 | 含义 |
|---|---|
| `scan:queued` | 任务已入队 |
| `scan:started` | worker 开始执行 |
| `scan:progress` | 进度更新（含 stage） |
| `scan:log` | stdout 逐行 |
| `scan:update` | 状态机迁移 |
| `scan:done` | 终态（success/failed/canceled） |
| `scan:error` | 入参/启动/解析错误 |

订阅方式见 [`frontend/src/composables/useWailsEvent.ts`](frontend/src/composables/useWailsEvent.ts)。

## Token 安全提示

云漏洞库 token 目前**明文存**在 `%APPDATA%/opensca-ui/config.json`，建议：

- 不要把整个 `opensca-ui` 目录加入公共仓库
- 仅使用本地漏洞库（`-db`）时可让 token 留空

## License

MIT