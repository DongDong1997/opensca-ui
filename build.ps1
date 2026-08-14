# Build wrapper for OpenSCA UI.
#
# 版本号解析（git tag 单一来源）：
#   - 传 -Version X.Y.Z（CI 从 vX.Y.Z tag 提取后传入）→ 权威发布版本。
#     会回写 wails.json 的 info.productVersion 并同步 frontend/package.json(+lock)，
#     NSIS 安装包元信息、输出文件名、前端构建（UI 里显示的版本号）全部用它。
#   - 不传 -Version → 回退用 wails.json 已提交的 productVersion（本地开发/手动构建）。
#
# 发布流程：
#   git tag -a v1.1.0 -m "..." && git push origin v1.1.0
#   CI 自动从 tag 提取版本 → ./build.ps1 -Version 1.1.0
#
# Output filenames produced:
#   build\bin\opensca-ui-<version>.exe                  (single exe)
#   build\bin\opensca-ui-<version>-amd64-installer.exe  (NSIS installer)
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File build.ps1
#   powershell -ExecutionPolicy Bypass -File build.ps1 -Version 1.1.0

param(
    # 发布版本（来自 git tag，形如 1.1.0）；不传则回退 wails.json 的 productVersion。
    [string]$Version
)

$ErrorActionPreference = 'Stop'

# 1. 版本号解析：-Version 优先（git tag 单一来源）；否则回退 wails.json
$wailsJsonPath = Join-Path $PSScriptRoot 'wails.json'
$cfg = Get-Content $wailsJsonPath -Raw | ConvertFrom-Json

if ($Version) {
    if ($Version -notmatch '^\d+\.\d+\.\d+$') {
        throw "非法版本号 '$Version'，需满足 X.Y.Z（如 1.1.0）"
    }
    if ($cfg.info.productVersion -ne $Version) {
        # 回写 wails.json：让 NSIS 安装包元信息 + 前端 vite define（UI 版本号）都用 tag 版本。
        $raw = Get-Content $wailsJsonPath -Raw
        $raw = [regex]::Replace($raw, '"productVersion"\s*:\s*"[^"]*"', ('"productVersion": "{0}"' -f $Version))
        [System.IO.File]::WriteAllText($wailsJsonPath, $raw)
        $cfg = Get-Content $wailsJsonPath -Raw | ConvertFrom-Json
        Write-Host "已把 wails.json productVersion 回写为 $Version" -ForegroundColor Yellow
    }
    # 同步前端元数据版本，避免 package.json 与 wails.json 漂移。
    # 不能用 ConvertFrom-Json 来回写 package-lock.json：lockfileVersion 3 的
    # packages[""] 是空字符串键，PowerShell 解析不了（PS7 报 "property whose name
    # is an empty string"，PS5.1 直接抛错）。两个文件都用文本替换，只改"根包"的版本：
    #   - package.json：顶层 "version"（2 空格缩进，全文件唯一）；
    #   - package-lock.json：顶层 "name" 后面的 "version"，以及 packages[""] 里的 "version"。
    # 依赖项自己的 "version" 不在上述位置，不会被误伤；其余内容逐字节保留。
    foreach ($p in @('frontend\package.json', 'frontend\package-lock.json')) {
        $path = Join-Path $PSScriptRoot $p
        if (-not (Test-Path $path)) { continue }
        $raw = Get-Content $path -Raw
        if ($p -like '*package-lock.json') {
            $raw = [regex]::Replace($raw, '("name": "[^"]*",\r?\n\s*"version":\s+)"[^"]*"', ('${1}"' + $Version + '"'))
        } else {
            $raw = [regex]::Replace($raw, '(?m)^(\s{2}"version":\s+)"[^"]*"', ('${1}"' + $Version + '"'))
        }
        [System.IO.File]::WriteAllText($path, $raw)
    }
    Write-Host "已同步 frontend/package.json 与 package-lock.json 版本为 $Version" -ForegroundColor DarkGray
}

$version = $cfg.info.productVersion
if (-not $version) {
    throw "wails.json 里找不到 info.productVersion；本地构建请先在 wails.json 填版本，或传 -Version"
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