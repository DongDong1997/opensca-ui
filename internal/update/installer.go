package update

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// InstallResult 描述一次下载并替换的结果。
type InstallResult struct {
	InstalledVersion string `json:"installedVersion"`
	BackupPath       string `json:"backupPath"`
	TargetPath       string `json:"targetPath"`
	Message          string `json:"message"`
}

// DownloadAndInstall 把 downloadURL 指向的 zip 下载并解压，把里面的可执行文件覆盖到 targetPath。
//
// 流程：下载到临时文件 → 解压到临时目录 → 找到目标 exe → 备份 targetPath 为 .bak → 复制新文件 → 删 .tmp。
//
// 失败时尽量回滚：若替换前失败不动原文件，若替换后失败保留 .bak 供用户手动恢复。
func DownloadAndInstall(downloadURL, targetPath string) (InstallResult, error) {
	if downloadURL == "" {
		return InstallResult{}, errors.New("downloadURL 为空")
	}
	if targetPath == "" {
		return InstallResult{}, errors.New("targetPath 为空")
	}

	res := InstallResult{TargetPath: targetPath}

	// 1. 下载到临时文件
	dlURLs := []string{
		downloadURL,
		"https://ghproxy.com/" + downloadURL,
		"https://mirror.ghproxy.com/" + downloadURL,
	}
	var zipPath string
	var lastErr error
	for _, u := range dlURLs {
		p, err := downloadZip(u)
		if err == nil {
			zipPath = p
			break
		}
		lastErr = err
		_ = p
	}
	if zipPath == "" {
		return res, fmt.Errorf("下载失败: %w", lastErr)
	}
	defer os.Remove(zipPath)

	// 2. 解压到临时目录
	tmpDir, err := os.MkdirTemp("", "opensca-cli-update-*")
	if err != nil {
		return res, err
	}
	defer os.RemoveAll(tmpDir)
	if err := unzip(zipPath, tmpDir); err != nil {
		return res, fmt.Errorf("解压失败: %w", err)
	}

	// 3. 在解压目录里找目标 exe
	exeName := "opensca-cli"
	if runtime.GOOS == "windows" {
		exeName = "opensca-cli.exe"
	}
	newExe, err := findFile(tmpDir, exeName)
	if err != nil {
		return res, fmt.Errorf("zip 内未找到 %s: %w", exeName, err)
	}

	// 4. 备份原文件
	if _, err := os.Stat(targetPath); err == nil {
		backup := targetPath + ".bak"
		// 已有 .bak 则覆盖
		_ = os.Remove(backup)
		if err := os.Rename(targetPath, backup); err != nil {
			// Rename 在跨卷时可能失败 → 退化为 copy+remove
			if err := copyFile(targetPath, backup); err != nil {
				return res, fmt.Errorf("备份原文件失败: %w", err)
			}
			_ = os.Remove(targetPath)
		}
		res.BackupPath = backup
	}

	// 5. 复制新文件到目标位置（覆盖）
	if err := copyFile(newExe, targetPath); err != nil {
		return res, fmt.Errorf("复制新文件失败: %w", err)
	}

	// 6. 设置可执行权限（非 Windows）
	if runtime.GOOS != "windows" {
		_ = os.Chmod(targetPath, 0o755)
	}

	res.Message = "更新成功"
	return res, nil
}

func downloadZip(url string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "opensca-ui/0.1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.CreateTemp("", "opensca-cli-*.zip")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func unzip(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		// 防 Zip Slip：确保目标路径在 dst 之下
		target := filepath.Join(dst, f.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dst)+string(os.PathSeparator)) && filepath.Clean(target) != filepath.Clean(dst) {
			return fmt.Errorf("非法 zip 路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		in, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			in.Close()
			out.Close()
			return err
		}
		in.Close()
		out.Close()
	}
	return nil
}

// findFile 在目录树中递归查找名为 name 的文件，返回第一个匹配。
func findFile(root, name string) (string, error) {
	lower := strings.ToLower(name)
	var found string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.ToLower(info.Name()) == lower {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return "", err
	}
	if found == "" {
		return "", errors.New("未找到")
	}
	return found, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
