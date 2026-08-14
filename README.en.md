<div align="center">

<img src="build/appicon.png" width="96" alt="OpenSCA UI" />

# 🛡️ OpenSCA UI

**English** | [简体中文](README.md)

A desktop GUI client for open-source Software Composition Analysis (SCA), built with [Wails](https://wails.io) wrapping [opensca-cli](https://github.com/XmirrorSecurity/OpenSCA-cli).
**Works out of the box — no separate CLI download needed.**

![Release](https://img.shields.io/github/v/release/DongDong1997/opensca-ui?style=flat-square&logo=github)
![Downloads](https://img.shields.io/github/downloads/DongDong1997/opensca-ui/total?style=flat-square)
![CI](https://img.shields.io/github/actions/workflow/status/DongDong1997/opensca-ui/build.yml?style=flat-square&logo=githubactions)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)
![Wails](https://img.shields.io/badge/Wails-v2-DF0000?style=flat-square)
![Vue](https://img.shields.io/badge/Vue-3.5-42b883?style=flat-square&logo=vuedotjs&logoColor=white)
![License](https://img.shields.io/github/license/DongDong1997/opensca-ui?style=flat-square)

</div>

**OpenSCA UI** is a desktop client for the open-source SCA tool [OpenSCA](https://opensca.xmirror.cn/). Pick a project directory or archive and start a scan — it automatically identifies the component inventory and detects known vulnerabilities (CVE/CWE), then outputs a readable report.

`opensca-cli` is bundled in, so the app is ready to scan right after installation. It supports concurrent scans, live logs, light/dark themes, and one-click CLI updates.

## ✨ Features

- 📁 **Multiple ways to scan**: pick a folder / archive, or drag & drop it onto the window
- 🚀 **Concurrent scans**: 3 workers by default, adjustable (1–10) in Settings
- 📜 **Live progress & logs**: streamed stdout output line by line, task state always visible
- 🔍 **Vulnerability visualization**: component inventory, severity, CVE/CWE, remediation advice; filtering + detail drawer
- 📊 **Report export**: HTML / JSON / Excel (generated natively by opensca-cli)
- 🌗 **Light / dark theme**
- 🔄 **CLI management**: bundled CLI, update checks, one-click download & replace
- ⚙️ **Rich settings**: CLI path, cloud / local vuln DB token, report directory, concurrency limit

## 📥 Installation

Download the latest release from the [Releases](https://github.com/DongDong1997/opensca-ui/releases) page:

| File | Description |
|---|---|
| `opensca-ui-<version>-amd64-installer.exe` | NSIS installer (recommended — configures the bundled CLI at install time) |
| `opensca-ui-<version>.exe` | Portable single-file executable |

> Both the installer and the portable build embed `opensca-cli` — open the app and scan, **no separate download**.
> The CLI defaults to the bundled one in the install directory; you can point it elsewhere under **Settings → CLI** (your choice is preserved).

## 🚀 Development

Prerequisites: **Go 1.25+**, **Node.js 20.19+ / 22.12+**, [Wails CLI](https://wails.io/docs/gettingstarted/installation) v2.

```bash
# 1. Fetch the bundled opensca-cli (internal/bundle/opensca-cli.exe, ~20 MB, first time only)
powershell -ExecutionPolicy Bypass -File scripts\fetch-cli.ps1

# 2. Dev mode: launches Vite dev server + Go backend automatically
wails dev
```

## 🔨 Build & Release

```powershell
# Single-file exe + NSIS installer (version is read from wails.json)
powershell -ExecutionPolicy Bypass -File build.ps1
```

A GitHub Actions workflow is set up — **pushing a `v*` tag** (e.g. `v1.0.0`) builds and publishes a Release automatically:

```bash
git tag -a v1.0.0 --cleanup=verbatim -m "Release v1.0.0" -m "## What's Changed

- Added: xxx
- Fixed: xxx"
git push origin v1.0.0
```

> The Release body is taken from the tag annotation message; `--cleanup=verbatim` preserves lines starting with `#` (Markdown headings).

## 📁 Project Structure

```
opensca-ui/
├── main.go                # Wails entry point
├── app.go                 # Frontend-bound methods (GetConfig / StartScan / CancelScan …)
├── wails.json             # Wails build config
├── build.ps1              # One-click build script (exe + NSIS installer)
├── internal/
│   ├── bundle/            # Bundled opensca-cli (go:embed, installed next to the app)
│   ├── config/            # Config management (%APPDATA%/opensca-ui/config.json)
│   ├── scanner/           # Scan task manager (worker pool + subprocess)
│   └── platform/          # Cross-platform paths
├── scripts/
│   └── fetch-cli.ps1      # Fetches the bundled opensca-cli
├── frontend/              # Vue 3 frontend
│   └── src/
│       ├── views/         # Welcome / Scan / Tasks / Report / Settings
│       ├── stores/        # Pinia: tasks / config / ui
│       ├── components/    # DropZone / TaskCard / LogViewer / VulnTable …
│       └── composables/   # useWailsEvent / useTaskStream / useTheme
└── build/                 # Wails assets (icon / NSIS scripts) and build output
```

## 🧱 Tech Stack

| Layer | Technology |
|---|---|
| Desktop framework | [Wails v2](https://wails.io) (Go + WebView2) |
| Backend | Go 1.25, spawns opensca-cli as a subprocess |
| Frontend | Vue 3.5 / TypeScript 5.6 / Vite 7 |
| UI library | Naive UI 2.40 (dark theme support) |
| State | Pinia 2.2 |
| Router | vue-router 4 (hash mode — no server for desktop apps) |

## 💾 Data Locations

| Content | Path |
|---|---|
| Config | `%APPDATA%\opensca-ui\config.json` |
| Bundled CLI | `install dir\opensca-cli.exe` (next to the app exe) |
| Task reports | `%APPDATA%\opensca-ui\reports\<taskID>.json` |
| Task logs | `%APPDATA%\opensca-ui\logs\<taskID>.log` |

## 🔒 Security Note

The cloud vuln DB token is currently stored in **plain text** in `config.json`:

- Do not make the directory containing your config publicly accessible
- If you only use the local vuln DB (`-db`), the token can be left empty

## 🤝 Acknowledgments

- [XmirrorSecurity/OpenSCA-cli](https://github.com/XmirrorSecurity/OpenSCA-cli) — the underlying scan engine
- [Wails](https://wails.io) — the desktop app framework

## 📄 License

[MIT](LICENSE) © 2026 Panda97
