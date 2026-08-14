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
#    只 prepend 实际存在的目录：本地开发机工具装在 D:\App\...；
#    GitHub Actions 里由 setup-go / setup-node / choco 提供（大多已在 PATH），
#    这里做兜底，避免拼出无效路径。
$toolDirs = @(
    'D:\App\NSIS', 'D:\App\Go\bin', 'D:\App\NodeJS',
    "$env:GOPATH\bin",
    'C:\Program Files (x86)\NSIS', 'C:\Program Files\NSIS'
)
foreach ($p in $toolDirs) {
    if ($p -and (Test-Path $p)) { $env:Path = "$p;$env:Path" }
}

# 2.5 校验内置 CLI：internal/bundle/opensca-cli.exe 必须是真实 exe（>1MB），
#     否则 go:embed 打包进去的是占位文件，用户装完仍要手动下载 CLI。
$bundleCli = Join-Path $PSScriptRoot 'internal\bundle\opensca-cli.exe'
if (-not (Test-Path $bundleCli) -or (Get-Item $bundleCli).Length -lt 1MB) {
    Write-Warning 'internal\bundle\opensca-cli.exe 不存在或疑似占位文件（<1MB）！'
    Write-Warning '构建出的安装包将不带内置 CLI，用户仍要手动下载 opensca-cli。'
    Write-Warning '修复: powershell -ExecutionPolicy Bypass -File scripts\fetch-cli.ps1'
}

# 3. 调 wails build
#    依次找 wails.exe：go env GOPATH\bin（CI/本机的 go install 落盘处）→
#    %USERPROFILE%\go\bin（老默认位置）→ PATH 里的 wails。
#    注意：CI runner 上 $env:GOPATH 可能是空（Go 内部用默认值），所以用 `go env` 解析。
$wailsCandidates = @()
$goCmd = Get-Command go -ErrorAction SilentlyContinue
if ($goCmd) {
    $gopath = (& $goCmd.Source env GOPATH) -join ''
    if ($gopath) { $wailsCandidates += (Join-Path $gopath 'bin\wails.exe') }
}
if ($env:USERPROFILE) { $wailsCandidates += (Join-Path $env:USERPROFILE 'go\bin\wails.exe') }

$wailsExe = $null
foreach ($candidate in $wailsCandidates) {
    if ($candidate -and (Test-Path $candidate)) { $wailsExe = $candidate; break }
}
if (-not $wailsExe) {
    $cmd = Get-Command wails -ErrorAction SilentlyContinue
    if ($cmd) { $wailsExe = $cmd.Source }
}
if (-not $wailsExe) {
    throw "找不到 wails.exe，请先执行: go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2"
}
Write-Host "使用 wails: $wailsExe" -ForegroundColor DarkGray

& $wailsExe build -clean -nsis -o "opensca-ui-$version.exe"
exit $LASTEXITCODE