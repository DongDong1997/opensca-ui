package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"opensca-ui/internal/bundle"
	"opensca-ui/internal/config"
	"opensca-ui/internal/history"
	"opensca-ui/internal/platform"
	"opensca-ui/internal/projectstore"
	"opensca-ui/internal/recent"
	"opensca-ui/internal/scanner"
	"opensca-ui/internal/update"
)

// App 是 Wails 绑定到前端的对象。
//
// 所有可被前端调用（import { ... } from '../wailsjs/go/main/App'）的方法都集中在这里。
type App struct {
	ctx     context.Context
	cfg     *config.Store
	recent  *recent.Store
	history *history.Store
	scanner *scanner.Manager
	ready   atomic.Bool
}

// NewApp 构造一个 App（不启动 scanner）。recent/history 可以为 nil。
func NewApp(cfg *config.Store, rec *recent.Store, hist *history.Store) *App {
	return &App{cfg: cfg, recent: rec, history: hist}
}

// startup 在 Wails 应用启动时调用。我们在这里初始化 scanner manager，
// 并把 Wails 的事件发射器注入进去。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// 未配置 CLI 时，用应用内置的 opensca-cli 作为默认（安装路径下，opensca-ui.exe 同目录）。
	// 已配置路径的用户保持原设置不动（"设置都保留"）。
	a.ensureBundledCli()
	emit := func(name string, data ...interface{}) {
		wruntime.EventsEmit(ctx, name, data...)
	}
	a.scanner = scanner.NewManager(a.cfg, a.recent, a.history, emit)
	a.scanner.Start(ctx)
	a.ready.Store(true)
}

// ensureBundledCli 让默认 CLI 路径 = 安装路径下的内置 opensca-cli。
//
// 流程：
//  1. bundle.Ensure() 保证安装路径（opensca-ui.exe 同目录）下存在 opensca-cli：
//     NSIS 安装器已在安装时装好；单文件 exe / dev 则从嵌入字节解包。
//     未打包（0 字节占位）时返回 ("", nil)，直接回退到老流程。
//  2. 用户手动配置过非空路径（CliPathManual=true）→ 不动，保留设置。
//  3. 其余情况（未配置 / 路径被清空 / 旧版遗留路径）→ 统一写入安装路径的
//     内置 CLI 并立即校验；校验不过说明内置文件有问题，清掉路径让前端
//     回欢迎页引导用户手动配置。
func (a *App) ensureBundledCli() {
	path, err := bundle.Ensure()
	if err != nil {
		log.Printf("解包内置 opensca-cli 失败: %v", err)
		return
	}
	if path == "" {
		return // 未打包内置 CLI，走原有"用户手动配置"流程
	}
	// 用户手动配置过非空路径：保留设置，不覆盖。
	if cfg := a.cfg.Get(); cfg.CliPathManual && strings.TrimSpace(cfg.CLIPath) != "" {
		return
	}
	// 先写路径再校验；校验失败就把路径清掉（CheckCli 只写 Valid/Version，不动 Path）。
	if err := a.cfg.SetCLIPath(path); err != nil {
		log.Printf("设置内置 CLI 路径失败: %v", err)
		return
	}
	info, _ := a.CheckCli(path)
	if !info.Valid {
		log.Printf("内置 opensca-cli 校验失败（%s），等待用户手动配置", info.Message)
		_ = a.cfg.SetCLIPath("")
	}
}

// shutdown 在退出前调用，确保日志/配置都落盘。
func (a *App) shutdown(ctx context.Context) {
	if a.scanner != nil {
		a.scanner.Stop()
	}
	_ = a.cfg.FlushNow()
}

// ----- 配置（前端调用） -----

// GetConfig 返回当前完整配置。
func (a *App) GetConfig() config.Config {
	return a.cfg.Get()
}

// GetConfigPath 返回配置文件的绝对路径，用于在前端展示给用户做诊断。
func (a *App) GetConfigPath() string {
	return a.cfg.Path()
}

// SetCliPath 更新 CLI 路径，同时触发 CheckCli 更新 CLIValid/CLIVersion。
// 这是用户在设置/欢迎页手动配置的入口：标记 CliPathManual=true，
// 后续启动 ensureBundledCli 不会再自动覆盖成安装路径的默认值。
func (a *App) SetCliPath(path string) (config.CliInfo, error) {
	if err := a.cfg.SetCLIPathManual(path); err != nil {
		return config.CliInfo{}, err
	}
	return a.CheckCli(path)
}

// CheckCli 验证指定路径的 CLI 可执行性，返回版本号。
func (a *App) CheckCli(path string) (config.CliInfo, error) {
	if strings.TrimSpace(path) == "" {
		_ = a.cfg.SetCLIStatus(false, "")
		return config.CliInfo{Path: path, Valid: false, Message: "路径为空"}, nil
	}
	if _, err := os.Stat(path); err != nil {
		_ = a.cfg.SetCLIStatus(false, "")
		return config.CliInfo{Path: path, Valid: false, Message: "文件不存在"}, nil
	}

	version, raw, err := probeVersion(path)
	if err != nil {
		_ = a.cfg.SetCLIStatus(false, "")
		return config.CliInfo{Path: path, Valid: false, Message: "无法识别 CLI 版本: " + err.Error(), RawOutput: raw}, nil
	}

	_ = a.cfg.SetCLIStatus(true, version)
	return config.CliInfo{Path: path, Valid: true, Version: version, RawOutput: raw}, nil
}

// probeVersion 尝试多种 CLI 调用形式以兼容 opensca-cli / cobra / urfave/cli 等常见实现。
//
// 顺序：
//  1. -version / --version / -v
//  2. version 子命令（无 - 前缀）
//  3. -h / --help（cobra 的 help 通常会在末尾/开头包含版本号）
//
// 只要 stdout 包含任一可识别的版本号（v1.2.3、1.2.3、v3.0.0-beta 之类）就算成功。
// 返回值是匹配到的版本号字符串。
func probeVersion(path string) (string, string, error) {
	attempts := [][]string{
		{"-version"},
		{"--version"},
		{"-v"},
		{"version"},
		{"-h"},
		{"--help"},
	}
	for _, args := range attempts {
		cmd := exec.Command(path, args...)
		hideWindow(cmd)
		out, _ := cmd.CombinedOutput() // exit code 非 0 也无所谓，只看输出
		text := strings.TrimSpace(string(out))
		if text == "" {
			continue
		}
		// 1) 优先找含 "version" 字样的行
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			lower := strings.ToLower(line)
			if strings.Contains(lower, "version") || strings.Contains(lower, "ver:") {
				if v, ok := extractVersion(line); ok {
					return v, text, nil
				}
			}
		}
		// 2) 退化：任意一行包含版本号
		for _, line := range strings.Split(text, "\n") {
			if v, ok := extractVersion(line); ok {
				return v, text, nil
			}
		}
	}
	return "", "", errors.New("无法识别 CLI 版本（请确认它是 opensca-cli 可执行文件）")
}

// extractVersion 从一行文本里提取版本号。
// 接受的格式：v1.2.3 / 1.2.3 / v3.0.0-rc1 / v3.0.0+build.123 等。
var versionRegex = regexp.MustCompile(`v?\d+\.\d+(\.\d+)?([\-+][\w\.]+)?`)

func extractVersion(line string) (string, bool) {
	m := versionRegex.FindString(line)
	if m == "" {
		return "", false
	}
	return m, true
}

// SetMaxConcurrent 设置并发上限（同时重启 scanner pool）。
func (a *App) SetMaxConcurrent(n int) error {
	if err := a.cfg.SetMaxConcurrent(n); err != nil {
		return err
	}
	if a.scanner != nil {
		a.scanner.Stop()
		a.scanner.Start(a.ctx)
	}
	return nil
}

// SetToken 更新云漏洞库 token。
func (a *App) SetToken(token string) error {
	return a.cfg.SetToken(token)
}

// SetLanguage 更新界面语言（zh-CN / en-US），立即落盘。
func (a *App) SetLanguage(lang string) error {
	return a.cfg.SetLanguage(lang)
}

// SetLocalDB 更新本地漏洞库路径。
func (a *App) SetLocalDB(path string) error {
	return a.cfg.SetLocalDB(path)
}

// TokenInfo 验证 token 的返回。
type TokenInfo struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
	// Source 表示用什么 URL 验证的（默认是 opensca.xmirror.cn 的云端接口），
	// UI 上展示给用户确认走的是官方云。
	Source string `json:"source"`
}

// openscaCloudURL 默认云漏洞库 URL（与 opensca-cli 内置默认一致）。
// 用户如果私有部署了 OpenSCA SaaS，得改这里；当前固定官方云。
const openscaCloudURL = "https://opensca.xmirror.cn"

// VerifyToken 验证云漏洞库 token 是否有效。
//
// 流程：调云端 `/oss-saas/api-v1/open-sca-client/aes-key?clientId=X&ossToken=<token>`，
// 成功（code==0）即 token 有效。这是 opensca-cli 实际拿 AES key 用的同一接口，
// 所以这里过了 → 实际扫描时也能用。
//
// 返回 TokenInfo 而不是 error：调用方根据 Valid 决定 UI 文案，
// 网络/DNS 错误也走 valid=false + message="无法连接..."，前端不用 try/catch。
func (a *App) VerifyToken(token string) (TokenInfo, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return TokenInfo{Valid: false, Message: "token 为空", Source: openscaCloudURL}, nil
	}

	// 16 个大写字母当 clientId，跟 opensca-cli 一致（验证用，合法性不重要）
	const clientID = "OPENSCAUICLIENTID"

	u := openscaCloudURL + "/oss-saas/api-v1/open-sca-client/aes-key"
	q := url.Values{}
	q.Set("clientId", clientID)
	q.Set("ossToken", token)
	full := u + "?" + q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet, full, nil)
	if err != nil {
		return TokenInfo{Valid: false, Message: "构造请求失败: " + err.Error(), Source: openscaCloudURL}, nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return TokenInfo{Valid: false, Message: "无法连接云端: " + err.Error(), Source: openscaCloudURL}, nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var parsed struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		// 非 JSON 响应（HTML 错误页等）：给出原文提示
		return TokenInfo{
			Valid:   false,
			Message: fmt.Sprintf("云端响应异常 (HTTP %d): %s", resp.StatusCode, truncateForMsg(string(body), 200)),
			Source:  openscaCloudURL,
		}, nil
	}

	if parsed.Code == 0 && parsed.Message == "success" {
		return TokenInfo{Valid: true, Message: "token 有效", Source: openscaCloudURL}, nil
	}
	return TokenInfo{
		Valid:   false,
		Message: fmt.Sprintf("云端拒绝：%s (code=%d)", parsed.Message, parsed.Code),
		Source:  openscaCloudURL,
	}, nil
}

// truncateForMsg 把超长响应截短，避免 message 一行塞太长。
func truncateForMsg(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// SetFolderReportLocation 设置"文件夹扫描"报告位置策略。
//
//   - useDefault=true:  项目本地 .opensca-ui/reports/（不可写回退 %APPDATA%）
//   - useDefault=false: 用 customPath；customPath 为空时回退 %APPDATA%
func (a *App) SetFolderReportLocation(useDefault bool, customPath string) error {
	return a.cfg.SetFolderReportLocation(useDefault, customPath)
}

// SetZipReportLocation 设置"压缩包扫描"报告位置策略。
//
//   - useDefault=true:  %APPDATA%/opensca-ui/reports/
//   - useDefault=false: 用 customPath；customPath 为空时回退 %APPDATA%
func (a *App) SetZipReportLocation(useDefault bool, customPath string) error {
	return a.cfg.SetZipReportLocation(useDefault, customPath)
}

// GetDefaultReportsPath 返回默认报告目录（%APPDATA%/opensca-ui/reports）。
// 用于前端在用户取消勾选时给输入框填默认值。
func (a *App) GetDefaultReportsPath() string {
	d, err := platform.ReportsDir()
	if err != nil {
		return ""
	}
	return d
}

// ----- 任务（前端调用） -----

// StartScanRequest 是 StartScan 的入参。
type StartScanRequest struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}

// StartScan 提交一个扫描任务，返回 taskID。
func (a *App) StartScan(req StartScanRequest) (string, error) {
	if !a.ready.Load() {
		return "", errors.New("服务尚未就绪")
	}
	return a.scanner.Submit(scanner.StartRequest{Path: req.Path, Label: req.Label})
}

// CancelScan 取消指定任务。
func (a *App) CancelScan(taskID string) error {
	if a.scanner == nil {
		return errors.New("scanner 未初始化")
	}
	return a.scanner.Cancel(taskID)
}

// ListTasks 返回所有任务摘要。
func (a *App) ListTasks() []scanner.TaskSummary {
	if a.scanner == nil {
		return nil
	}
	return a.scanner.ListSummaries()
}

// GetTask 返回任务详情。返回指针避免对含 sync.Mutex 的 Task 做值拷贝。
func (a *App) GetTask(taskID string) (*scanner.Task, error) {
	if a.scanner == nil {
		return nil, errors.New("scanner 未初始化")
	}
	t, ok := a.scanner.Get(taskID)
	if !ok {
		return nil, fmt.Errorf("任务不存在: %s", taskID)
	}
	return t, nil
}

// GetTaskLogs 从指定 offset 读取任务日志（按行）。
func (a *App) GetTaskLogs(taskID string, offset int) (string, error) {
	if a.scanner == nil {
		return "", errors.New("scanner 未初始化")
	}
	return a.scanner.Logs(taskID, offset)
}

// GetTaskResult 解析任务报告。
func (a *App) GetTaskResult(taskID string) (scanner.Report, error) {
	if a.scanner == nil {
		return scanner.Report{}, errors.New("scanner 未初始化")
	}
	r, err := a.scanner.Result(taskID)
	if err != nil {
		return scanner.Report{}, err
	}
	return *r, nil
}

// DeleteTask 删除任务及其产物。
func (a *App) DeleteTask(taskID string) error {
	if a.scanner == nil {
		return errors.New("scanner 未初始化")
	}
	return a.scanner.Delete(taskID)
}

// ----- UI 辅助（前端调用） -----

// PickDirectory 弹出原生目录选择对话框。
func (a *App) PickDirectory() (string, error) {
	return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "选择项目目录",
	})
}

// PickZip 弹出原生文件选择对话框。
func (a *App) PickZip() (string, error) {
	return wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "选择项目压缩包",
		Filters: []wruntime.FileFilter{
			{DisplayName: "压缩包 (*.zip;*.tar.gz)", Pattern: "*.zip;*.tar.gz"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
}

// PickExecutable 弹出原生文件选择对话框，过滤可执行文件（用于选择 opensca-cli）。
func (a *App) PickExecutable() (string, error) {
	return wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "选择 opensca-cli",
		Filters: []wruntime.FileFilter{
			{DisplayName: "可执行文件 (*.exe)", Pattern: "*.exe"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
}

// OpenInFolder 用系统默认浏览器打开 URL（兼容 file:// 协议）。
//
// 用 cmd /c start "" 兜底，BrowserOpenURL 在某些 Windows 版本对 file:// 不响应。
func (a *App) OpenInFolder(url string) error {
	if url == "" {
		return errors.New("url 为空")
	}
	switch runtime.GOOS {
	case "windows":
		// rundll32 url.dll,FileProtocolHandler 是 Windows 打开任意 URL 的最稳办法，
		// 包括 file://、https:// 等所有协议。
		cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url)
		hideWindow(cmd)
		return cmd.Start()
	default:
		wruntime.BrowserOpenURL(a.ctx, url)
		return nil
	}
}

// ShowItemInFolder 在文件管理器中显示指定文件。
//
// Windows 上 explorer.exe /select,<path> 在以下情况会失败并打开"此电脑"：
//   - 路径里有空格但没用引号包裹（少数 Windows 版本）
//   - 文件在隐藏目录（.开头）下
//   - 路径用了网络盘符或者 UNC 解析失败
// 这里始终用反斜杠（Windows 习惯）+ 引号包裹 + 同步 Run 让 fallback 可触发。
func (a *App) ShowItemInFolder(path string) error {
	if path == "" {
		return errors.New("路径为空")
	}
	switch runtime.GOOS {
	case "windows":
		// 用反斜杠、引号包裹，确保路径里有空格也 OK
		// explorer /select,"C:\path\to\file.ext"
		selectArg := "/select," + path
		cmd := exec.Command("explorer.exe", selectArg)
		hideWindow(cmd)
		if err := cmd.Start(); err != nil {
			return exec.Command("explorer.exe", filepath.Dir(path)).Start()
		}
		// explorer.exe 启动后即使进程退出也会保持窗口；不需要 Wait
		go func() { _ = cmd.Wait() }()
		return nil
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	default:
		return exec.Command("xdg-open", filepath.Dir(path)).Start()
	}
}

// CheckCliUpdate 查询 GitHub 上的最新 release，并与当前 CLI 版本比较。
// currentPath 可以为空，仅用于在错误信息里提示用户。
func (a *App) CheckCliUpdate(currentPath string) (update.Info, error) {
	c := a.cfg.Get()
	currentVersion := c.CLIVersion
	info, err := update.CheckUpdate(currentVersion)
	if err != nil && currentPath != "" {
		info.Message = info.Message + "（当前 CLI: " + currentPath + "）"
	}
	return info, err
}

// DownloadAndInstallCliUpdate 下载 release zip，解压后把 opensca-cli.exe 覆盖到 targetPath。
// targetPath 必须是当前 CLI 的可执行文件路径。
func (a *App) DownloadAndInstallCliUpdate(downloadURL, targetPath string) (update.InstallResult, error) {
	res, err := update.DownloadAndInstall(downloadURL, targetPath)
	if err != nil {
		return res, err
	}
	// CLI 换新后，重放内置辅助文件（config.json / db-demo.json），保证本地漏洞库
	// 来源不因换 exe 丢失。release zip 里只有 exe、没有 config，所以只能以内嵌版
	// 为基准；bundle.RefreshAux 只覆盖"缺失/损坏"的 config，用户自配的本地源保留。
	// 仅当目标是"安装目录里的内置 CLI"才做——用户手动指定的外部 CLI 目录不动。
	if dir, aerr := bundle.AppDir(); aerr == nil &&
		strings.EqualFold(filepath.Clean(filepath.Dir(targetPath)), filepath.Clean(dir)) {
		if aerr := bundle.RefreshAux(dir); aerr != nil {
			log.Printf("更新后刷新内置 config.json 失败: %v", aerr)
		}
	}
	// 更新完重新探测一次，让顶栏状态立刻刷新
	if _, cerr := a.CheckCli(targetPath); cerr != nil {
		res.Message += "（但重新验证失败：" + cerr.Error() + "）"
	}
	return res, nil
}

// OpenReleasePage 在系统浏览器里打开指定 URL。
func (a *App) OpenReleasePage(url string) error {
	if url == "" {
		return errors.New("URL 为空")
	}
	wruntime.BrowserOpenURL(a.ctx, url)
	return nil
}

// ----- 最近项目（首页用） -----

// GetRecentProjects 返回最近打开的项目列表（LastAt 倒序）。
func (a *App) GetRecentProjects() []recent.Entry {
	if a.recent == nil {
		return []recent.Entry{}
	}
	return a.recent.List()
}

// AddRecentProject 手动记录一次（HomeView 在点击"加入最近"时调）。
func (a *App) AddRecentProject(path, label string) error {
	if a.recent == nil {
		return errors.New("最近项目存储未初始化")
	}
	return a.recent.Add(path, label)
}

// RemoveRecentProject 从最近列表中移除一项。
func (a *App) RemoveRecentProject(path string) error {
	if a.recent == nil {
		return errors.New("最近项目存储未初始化")
	}
	return a.recent.Remove(path)
}

// ClearRecentProjects 清空最近列表。
func (a *App) ClearRecentProjects() error {
	if a.recent == nil {
		return errors.New("最近项目存储未初始化")
	}
	return a.recent.Clear()
}

// ----- 项目本地扫描记录 -----

// GetProjectScanHistory 读取项目目录下 .opensca-ui/scans.json。
// 入口：扫描过的项目（容易在 API 拼出 path）。
// 文件不存在返回空列表，不报错。
func (a *App) GetProjectScanHistory(projectPath string) []projectstore.Entry {
	list, err := projectstore.LoadHistory(projectPath)
	if err != nil {
		return []projectstore.Entry{}
	}
	return list
}

// GetProjectLocalDir 返回项目本地扫描目录路径（如果可写）。
// 仅用于前端诊断展示。
func (a *App) GetProjectLocalDir(projectPath string) string {
	if projectPath == "" {
		return ""
	}
	if !projectstore.IsUsable(projectPath) {
		return ""
	}
	return projectstore.ScanPath(projectPath)
}

// OpenProjectFolder 在文件管理器里打开项目目录（不进入 .opensca-ui）。
func (a *App) OpenProjectFolder(projectPath string) error {
	if projectPath == "" {
		return errors.New("项目路径为空")
	}
	return a.ShowItemInFolder(projectPath)
}

// hideWindow 让子进程不弹出黑色控制台（Windows）。
func hideWindow(cmd *exec.Cmd) {
	if runtime.GOOS != "windows" {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// CREATE_NO_WINDOW = 0x08000000
	cmd.SysProcAttr.CreationFlags |= 0x08000000
}