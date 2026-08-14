package platform

import (
	"os"
	"path/filepath"
)

// AppDataDir 返回 %APPDATA%/opensca-ui（跨平台：os.UserConfigDir()）。
// 首次调用时会创建目录。
func AppDataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "opensca-ui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// ReportsDir 返回报告目录（%APPDATA%/opensca-ui/reports）。
func ReportsDir() (string, error) {
	root, err := AppDataDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(root, "reports")
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	return d, nil
}

// LogsDir 返回日志目录。
func LogsDir() (string, error) {
	root, err := AppDataDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(root, "logs")
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	return d, nil
}