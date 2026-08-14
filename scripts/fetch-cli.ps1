# fetch-cli.ps1 — 下载 opensca-cli 的 windows-amd64 版并放入 internal/bundle/，
# 作为 OpenSCA UI 的内置 CLI（go:embed 打包进应用，用户无需再单独下载）。
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File scripts\fetch-cli.ps1            # 拉最新版
#   powershell -ExecutionPolicy Bypass -File scripts\fetch-cli.ps1 -Version v3.0.11  # 指定版本
#
# 产物：internal\bundle\opensca-cli.exe（约 20MB）
# 说明：构建发布包前请确保该文件是真实 exe（非 0 字节占位）。

param([string]$Version = '')

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$dstDir = Join-Path $root 'internal\bundle'
$dst = Join-Path $dstDir 'opensca-cli.exe'

# 1. 解析目标版本 → release 信息
$apiUrl = "https://api.github.com/repos/XmirrorSecurity/OpenSCA-cli/releases/latest"
if ($Version) {
    $apiUrl = "https://api.github.com/repos/XmirrorSecurity/OpenSCA-cli/releases/tags/$Version"
}

$release = $null
foreach ($prefix in @('', 'https://mirror.ghproxy.com/')) {
    $url = $prefix + $apiUrl
    try {
        Write-Host "查询 release: $url" -ForegroundColor DarkGray
        $release = Invoke-RestMethod -Uri $url -Headers @{ 'User-Agent' = 'opensca-ui/0.1.0' } -TimeoutSec 30
        if ($release.tag_name) { break }
    } catch {
        Write-Host "  失败: $($_.Exception.Message)" -ForegroundColor DarkGray
    }
}
if (-not $release -or -not $release.tag_name) {
    throw "无法获取 release 信息，请检查网络（可指定 -Version 重试）"
}

# 2. 找 windows-amd64 的 zip 资产
$asset = $release.assets | Where-Object {
    $_.name -match 'windows-amd64' -and $_.name -like '*.zip'
} | Select-Object -First 1
if (-not $asset) {
    $asset = $release.assets | Where-Object { $_.name -like '*.zip' } | Select-Object -First 1
}
if (-not $asset) {
    throw "release 里没找到 zip 资产"
}
Write-Host "版本: $($release.tag_name)  资产: $($asset.name) ($([math]::Round($asset.size / 1MB, 1)) MB)" -ForegroundColor Cyan

# 3. 下载 zip
$zip = Join-Path $env:TEMP "opensca-cli-$([guid]::NewGuid().ToString('N')).zip"
Write-Host '下载中…'
$dlUrl = $null
foreach ($prefix in @('', 'https://mirror.ghproxy.com/')) {
    $url = $prefix + $asset.browser_download_url
    try {
        Invoke-WebRequest -Uri $url -OutFile $zip -Headers @{ 'User-Agent' = 'opensca-ui/0.1.0' } -TimeoutSec 300
        $dlUrl = $url
        break
    } catch {
        Write-Host "  失败: $($_.Exception.Message)" -ForegroundColor DarkGray
    }
}
if (-not (Test-Path $zip)) {
    throw "下载失败，请手动下载 $($asset.browser_download_url)"
}

# 4. 解压并定位 opensca-cli.exe
$extract = Join-Path $env:TEMP "opensca-cli-extract-$([guid]::NewGuid().ToString('N'))"
Expand-Archive -Path $zip -DestinationPath $extract -Force
$exe = Get-ChildItem -Path $extract -Recurse -Filter 'opensca-cli.exe' | Select-Object -First 1
if (-not $exe) {
    Remove-Item $extract -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item $zip -Force -ErrorAction SilentlyContinue
    throw 'zip 里没找到 opensca-cli.exe'
}

# 5. 覆盖写入 internal/bundle/
New-Item -ItemType Directory -Path $dstDir -Force | Out-Null
Copy-Item -Path $exe.FullName -Destination $dst -Force
Remove-Item $extract -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item $zip -Force -ErrorAction SilentlyContinue

Write-Host "已放入内置 CLI: $dst ($([math]::Round((Get-Item $dst).Length / 1MB, 1)) MB)" -ForegroundColor Green
