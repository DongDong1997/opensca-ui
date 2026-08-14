# Build wrapper for OpenSCA UI.
#
# Reads productVersion from wails.json (single source of truth) and invokes
# `wails build -clean -nsis -o opensca-ui-<version>.exe`. Both the bare exe
# and the NSIS installer filename pick up the version automatically.
#
# Output filenames produced:
#   build\bin\opensca-ui-<version>.exe                  (single exe)
#   build\bin\opensca-ui-<version>-amd64-installer.exe  (NSIS installer)
#
# To release: bump productVersion in wails.json — both filenames follow.
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File build.ps1

$ErrorActionPreference = 'Stop'

# 1. 读版本号（来自 wails.json 的 info.productVersion）
$wailsJsonPath = Join-Path $PSScriptRoot 'wails.json'
$cfg = Get-Content $wailsJsonPath -Raw | ConvertFrom-Json
$version = $cfg.info.productVersion
if (-not $version) {
    throw "wails.json 里找不到 info.productVersion"
}
Write-Host "Building OpenSCA UI v$version ..." -ForegroundColor Cyan

# 2. 把 Go / Node / NSIS 加到 PATH（让 Wails CLI 能找到编译器和 makensis）
$env:Path = 'D:\App\NSIS;D:\App\Go\bin;D:\App\NodeJS;' + $env:Path

# 2.5 校验内置 CLI：internal/bundle/opensca-cli.exe 必须是真实 exe（>1MB），
#     否则 go:embed 打包进去的是占位文件，用户装完仍要手动下载 CLI。
$bundleCli = Join-Path $PSScriptRoot 'internal\bundle\opensca-cli.exe'
if (-not (Test-Path $bundleCli) -or (Get-Item $bundleCli).Length -lt 1MB) {
    Write-Warning 'internal\bundle\opensca-cli.exe 不存在或疑似占位文件（<1MB）！'
    Write-Warning '构建出的安装包将不带内置 CLI，用户仍要手动下载 opensca-cli。'
    Write-Warning '修复: powershell -ExecutionPolicy Bypass -File scripts\fetch-cli.ps1'
}

# 3. 调 wails build
$wailsExe = 'C:\Users\hdec\go\bin\wails.exe'
if (-not (Test-Path $wailsExe)) {
    throw "找不到 $wailsExe，请先执行: go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2"
}

& $wailsExe build -clean -nsis -o "opensca-ui-$version.exe"
exit $LASTEXITCODE