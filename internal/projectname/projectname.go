// Package projectname 提供从扫描路径推导"项目名"的共享工具。
//
// 项目名是稳定标识（同一目录/压缩包每次推导结果相同），用来把多次扫描记录
// 关联到同一个项目。规则：
//   - 目录   → 取 basename
//   - 压缩包 → basename 去扩展名（.tar.gz / .zip / .tar 等）
//
// 历史记录和最近项目列表都依赖此函数推导 projectName，因此放在独立的
// 小包里，避免 recent → scanner / scanner → recent 的循环依赖。
package projectname

import (
	"os"
	"path/filepath"
	"strings"
)

// Derive 从路径推导项目名。空字符串或无效路径返回空串，调用方负责回退。
func Derive(p string) string {
	if strings.TrimSpace(p) == "" {
		return ""
	}
	info, err := os.Stat(p)
	isDir := err == nil && info.IsDir()
	base := filepath.Base(p)
	if !isDir {
		low := strings.ToLower(base)
		// .tar.gz 这种双扩展名优先处理
		if strings.HasSuffix(low, ".tar.gz") {
			base = base[:len(base)-len(".tar.gz")]
		} else if ext := filepath.Ext(base); ext != "" {
			base = strings.TrimSuffix(base, ext)
		}
	}
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}