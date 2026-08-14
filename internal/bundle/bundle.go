package bundle

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// 内置 opensca-cli 可执行文件。
//
// 构建前把真实 opensca-cli.exe 放到本目录（internal/bundle/opensca-cli.exe），
// 它会经 go:embed 打进应用二进制，同时被 NSIS 安装器装到 $INSTDIR（安装路径）。
// 这样用户装完软件即可扫描，无需再单独下载 opensca-cli。
//
// 若目录里只有 0 字节占位文件（开发期还没放真实 exe），go build 仍能通过，
// Bytes() 返回 (nil, false)，应用回退到"用户手动配置路径"的老流程。
//
//go:embed opensca-cli.exe
var cliExe embed.FS

// embeddedName 是嵌入文件的固定名（Windows 可执行文件）。
const embeddedName = "opensca-cli.exe"

// Name 返回可执行文件名（跨平台：Windows 带 .exe 后缀）。
func Name() string {
	if runtime.GOOS == "windows" {
		return "opensca-cli.exe"
	}
	return "opensca-cli"
}

// Bytes 返回嵌入的 CLI 内容。未打包（0 字节占位）时返回 (nil, false)。
func Bytes() ([]byte, bool) {
	data, err := cliExe.ReadFile(embeddedName)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

// AppDir 返回应用可执行文件所在目录，即"安装路径"（opensca-ui.exe 同目录）。
func AppDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

// Path 返回内置 CLI 解包/安装后的完整路径（安装路径下）。
func Path() (string, error) {
	d, err := AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, Name()), nil
}

// Ensure 保证安装路径下存在 opensca-cli，返回完整路径。
//
// 两条来源：
//   - NSIS 安装器已在安装时把 internal/bundle/opensca-cli.exe 装进 $INSTDIR
//     （安装器有管理员权限，能写 Program Files），此时文件已存在，直接返回。
//   - 单文件 exe / dev 运行没有安装器：把 go:embed 的字节解包到应用同目录兜底。
//
// 决策：
//   - 未打包（占位文件）：返回 ("", nil)，调用方走原有"手动配置"流程。
//   - 目标已存在且非空：跳过解包。刻意为之 —— 用户通过"下载并替换"更新过的
//     那份要保留，不能每次启动都被旧的内置版覆盖。
//   - 目标存在但是 0 字节（半截文件）：覆盖重写。
//
// 解包写盘用 tmp + rename 原子替换，避免应用中途退出留下损坏的半截文件。
func Ensure() (string, error) {
	data, ok := Bytes()
	if !ok {
		return "", nil
	}
	dst, err := Path()
	if err != nil {
		return "", err
	}
	if fi, err := os.Stat(dst); err == nil && fi.Size() > 0 {
		return dst, nil
	}
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return "", fmt.Errorf("写入内置 CLI 失败: %w", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("解包内置 CLI 失败: %w", err)
	}
	return dst, nil
}
