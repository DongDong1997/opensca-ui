package bundle

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
//go:embed opensca-cli.exe config.json db-demo.json
var cliFS embed.FS

// 嵌入文件的固定名（Windows 可执行文件 + 随包辅助文件）。
// 注意：opensca-cli v3.x 只认自身运行目录下名为 config.json 的文件，
// 所以辅助配置必须用这个名字解包，换成别的名字 CLI 会当成默认配置。
const (
	embeddedName       = "opensca-cli.exe"
	embeddedConfigName = "config.json"
	embeddedDBName     = "db-demo.json"
)

// auxFileNames 是随内置 CLI 一起解包的辅助文件。
//
// opensca-cli v3.x 会从自身运行目录自动读取 config.json；这份 config 里
// origin.json 指向 db-demo.json（本地漏洞库），保证内置 CLI 不配 token
// 也能查到漏洞、不再报 "not config vuln database origin"。
var auxFileNames = []string{embeddedConfigName, embeddedDBName}

// Name 返回可执行文件名（跨平台：Windows 带 .exe 后缀）。
func Name() string {
	if runtime.GOOS == "windows" {
		return "opensca-cli.exe"
	}
	return "opensca-cli"
}

// Bytes 返回嵌入的 CLI 内容。未打包（0 字节占位）时返回 (nil, false)。
func Bytes() ([]byte, bool) {
	data, err := cliFS.ReadFile(embeddedName)
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

// writeFile 把 go:embed 里的 name 解包到 dir/name，覆盖已存在文件。
// 写盘用 tmp + rename 原子替换，避免应用中途退出留下损坏的半截文件。
func writeFile(name, dir string) error {
	data, err := cliFS.ReadFile(name)
	if err != nil || len(data) == 0 {
		return nil
	}
	dst := filepath.Join(dir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return fmt.Errorf("写入内置文件 %s 失败: %w", name, err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("解包内置文件 %s 失败: %w", name, err)
	}
	return nil
}

// ensureFile 与 writeFile 相同，但目标已存在且非空时跳过
// （刻意为之：用户"下载并替换"更新过的版本要保留，
// 不能每次启动都被旧的内置版覆盖）。
func ensureFile(name, dir string) error {
	dst := filepath.Join(dir, name)
	if fi, err := os.Stat(dst); err == nil && fi.Size() > 0 {
		return nil
	}
	return writeFile(name, dir)
}

// config.json 里"可用本地漏洞库源"的检测。
// opensca-cli 只认运行目录下名为 config.json 的文件；origin 里 json/dsn 全空时
// 会报 "not config vuln database origin"。用正则扫文本而非 JSON 解析，
// 因为内置 config 是 JSON5（带中文注释），标准库解析不了；只匹配非空值，
// 空值 / 缺字段都算"没配"。经手测：内嵌版（json=db-demo.json）→ true；
// origin.json 清空 / 默认模板 → false；自定义 json 或 mysql dsn → true。
var (
	reJSONOrigin = regexp.MustCompile(`"json"\s*:\s*"[^\s"]+`)
	reDSNOrigin  = regexp.MustCompile(`"dsn"\s*:\s*"[^\s"]+`)
)

func configHasLocalOrigin(text string) bool {
	return reJSONOrigin.MatchString(text) || reDSNOrigin.MatchString(text)
}

// RefreshAux 在 CLI 更新成功后重放内置辅助文件，保证本地漏洞库配置不丢。
//
// 覆盖条件只限"缺失 / 损坏"，绝不破坏用户自己的配置：
//   - config.json 缺失、为空，或 origin 里没有可用本地源（json/dsn 全空，
//     会触发 "not config vuln database origin"）→ 覆盖为内置版（origin.json
//     指向 db-demo.json），并把 db-demo.json 一起放好。
//   - config.json 已有可用本地源（用户自配了本地库）→ 原样保留，不碰。
//
// 与 Ensure 的区别：Ensure 启动时跑、对已存在非空文件一律跳过；这里在
// "CLI 换新后"立即再跑一次，且能修复"存在但损坏"的 config.json。
func RefreshAux(dir string) error {
	if _, ok := Bytes(); !ok {
		return nil
	}
	cfgPath := filepath.Join(dir, embeddedConfigName)
	if data, err := os.ReadFile(cfgPath); err == nil && len(data) > 0 && configHasLocalOrigin(string(data)) {
		return nil // 已有可用本地源，保留用户版本
	}
	for _, name := range auxFileNames {
		if err := writeFile(name, dir); err != nil {
			return err
		}
	}
	return nil
}

// Ensure 保证安装路径下存在 opensca-cli 及其辅助文件，返回 CLI 完整路径。
//
// 两条来源：
//   - NSIS 安装器已在安装时把 internal/bundle/opensca-cli.exe 装进 $INSTDIR
//     （安装器有管理员权限，能写 Program Files），此时文件已存在，直接跳过解包。
//   - 单文件 exe / dev 运行没有安装器：把 go:embed 的字节解包到应用同目录兜底。
//
// 决策：
//   - 未打包（占位文件）：返回 ("", nil)，调用方走原有"手动配置"流程。
//   - 目标已存在且非空：跳过解包。刻意为之 —— 用户通过"下载并替换"更新过的
//     那份要保留，不能每次启动都被旧的内置版覆盖。
//   - 目标存在但是 0 字节（半截文件）：覆盖重写。
//
// 辅助文件（config.json / db-demo.json）同样解包到 CLI 同目录：
// opensca-cli v3.x 从自身运行目录自动读 config.json，缺了会报
// "not config vuln database origin"。
func Ensure() (string, error) {
	if _, ok := Bytes(); !ok {
		return "", nil
	}
	dst, err := Path()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(dst)
	// CLI 本体
	if err := ensureFile(embeddedName, dir); err != nil {
		return "", err
	}
	// 随包辅助文件
	for _, name := range auxFileNames {
		if err := ensureFile(name, dir); err != nil {
			return "", err
		}
	}
	// 上面的 ensureFile 对"存在但损坏"的 config.json（origin 没配本地源）
	// 会跳过不处理，这里补一刀：缺失/损坏才覆盖，有效配置保留。
	if err := RefreshAux(dir); err != nil {
		return "", err
	}
	return dst, nil
}
