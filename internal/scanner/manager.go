package scanner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"opensca-ui/internal/config"
	"opensca-ui/internal/history"
	"opensca-ui/internal/platform"
	"opensca-ui/internal/projectname"
	"opensca-ui/internal/projectstore"
	"opensca-ui/internal/recent"
)

// Emitter 把事件转发给 Wails runtime（由 App 在 startup 时注入）。
//
// 用 interface{} 数组而不是结构体，是因为 Wails 的 EventsEmit 接受可变参数。
type Emitter func(name string, data ...interface{})

// Manager 全局扫描任务管理器。
type Manager struct {
	cfg     *config.Store
	recent  *recent.Store
	history *history.Store
	emit    Emitter
	mu      sync.RWMutex
	tasks   map[string]*Task
	queue   chan *Task
	workers int
	rootCtx context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewManager 构造 Manager（不启动 worker）。
// recent / history 可以为 nil（不记录历史）；生产环境一定要传。
func NewManager(cfg *config.Store, rec *recent.Store, hist *history.Store, emit Emitter) *Manager {
	m := &Manager{
		cfg:     cfg,
		recent:  rec,
		history: hist,
		emit:    emit,
		tasks:   make(map[string]*Task),
		queue:   make(chan *Task, 64),
	}
	// 启动时把历史里"还在运行中"的记录强制改为 canceled（启动时 worker 都不在了）。
	// 这样前端 TasksView 不会看到一堆幽灵 Running。
	if hist != nil {
		for _, e := range hist.List() {
			if e.Status == string(StatusRunning) || e.Status == string(StatusPending) {
				e.Status = string(StatusCanceled)
				e.Error = "上次应用未正常退出"
				_ = hist.Upsert(e)
			}
		}
	}
	return m
}

// Start 启动 N 个 worker goroutine。N 来自 cfg.MaxConcurrent。
func (m *Manager) Start(parent context.Context) {
	m.rootCtx, m.cancel = context.WithCancel(parent)
	c := m.cfg.Get()
	n := c.MaxConcurrent
	if n < 1 {
		n = 1
	}
	m.workers = n
	for i := 0; i < n; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}
}

// Stop 停止所有 worker，cancel 所有运行中任务。
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	close(m.queue)
	m.wg.Wait()
}

func (m *Manager) worker(id int) {
	defer m.wg.Done()
	for t := range m.queue {
		m.runOne(t)
	}
}

// Submit 入队一个新任务。返回 taskID 和错误。
func (m *Manager) Submit(req StartRequest) (string, error) {
	cfg := m.cfg.Get()
	if strings.TrimSpace(cfg.CLIPath) == "" {
		return "", fmt.Errorf("未配置 opensca-cli 路径，请先到设置页配置")
	}
	if _, err := os.Stat(cfg.CLIPath); err != nil {
		return "", fmt.Errorf("opensca-cli 不存在: %s", cfg.CLIPath)
	}
	if strings.TrimSpace(req.Path) == "" {
		return "", fmt.Errorf("扫描路径不能为空")
	}
	if _, err := os.Stat(req.Path); err != nil {
		return "", fmt.Errorf("扫描路径不可访问: %s", req.Path)
	}

	id := newID()

	// 项目名：从路径直接推导（目录取 basename；压缩包去扩展名）。
	// 作为记录的绑定键；同时写入 Task / 历史 / 项目本地索引。
	projectName := projectname.Derive(req.Path)

	// 根据扫描路径类型（目录/zip）+ 用户配置，决定 reports/ 与 logs/ 的根目录。
	rootDir, name, err := m.resolveArtifactsRoot(req.Path)
	if err != nil {
		return "", err
	}
	reportDir := filepath.Join(rootDir, "reports")
	logDir := filepath.Join(rootDir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", fmt.Errorf("创建日志目录失败: %w", err)
	}
	reportPath, htmlPath, err := projectstore.MakeReportPaths(reportDir, name, time.Now())
	if err != nil {
		return "", err
	}
	logFilePath := filepath.Join(logDir, id+".log")

	t := &Task{
		ID:          id,
		Label:       strings.TrimSpace(req.Label),
		ProjectName: projectName,
		Path:        req.Path,
		Status:      StatusPending,
		Progress:    0,
		Stage:       "等待中",
		ReportPath:  reportPath,
		HTMLPath:    htmlPath,
		LogPath:     logFilePath,
	}

	m.mu.Lock()
	m.tasks[id] = t
	m.mu.Unlock()

	// 记录到最近项目（失败也无所谓，不影响扫描本身）
	if m.recent != nil {
		displayName := firstNonEmpty(t.Label, t.ProjectName)
		_ = m.recent.Add(t.Path, displayName)
	}
	// 写入历史（pending 状态），重启后 ListSummaries 能看到
	m.saveHistory(t)

	m.emit("scan:queued", map[string]any{"taskID": id, "projectName": projectName})
	m.emit("scan:update", map[string]any{"taskID": id, "status": string(StatusPending)})

	select {
	case m.queue <- t:
		return id, nil
	default:
		m.failTask(t, fmt.Errorf("队列已满"))
		return id, nil
	}
}

// resolveArtifactsRoot 根据扫描路径类型（目录/zip）+ 用户配置，决定 reports/ 与 logs/ 的根目录。
//
// 返回值：
//   - rootDir: 同时包含 reports/ 和 logs/ 子目录的根
//   - name:    报告文件名前缀（目录扫描=文件夹名，zip 扫描=zip 去后缀）
//   - err:     路径无法访问 / 配置的自定义路径无法创建时报错
//
// 决策表（FolderReportUseDefault 默认 true）：
//
//	扫描类型 | useDefault | 结果
//	---------|------------|----------------------------------------
//	目录     | true       | 项目本地 .opensca-ui（不可写→ AppData）
//	目录     | false      | FolderReportCustomPath（空→ AppData）
//	zip      | true       | AppData
//	zip      | false      | ZipReportCustomPath（空→ AppData）
func (m *Manager) resolveArtifactsRoot(scanPath string) (rootDir, name string, err error) {
	info, err := os.Stat(scanPath)
	if err != nil {
		return "", "", fmt.Errorf("扫描路径不可访问: %s", scanPath)
	}

	appDataRoot, err := platform.AppDataDir()
	if err != nil {
		return "", "", err
	}
	appArtifactsRoot := appDataRoot

	cfg := m.cfg.Get()

	if info.IsDir() {
		// 文件夹扫描
		name = filepath.Base(scanPath)
		if cfg.FolderReportUseDefault {
			// 优先项目本地
			if projectstore.IsUsable(scanPath) {
				_ = projectstore.Ensure(scanPath)
				return projectstore.ScanPath(scanPath), name, nil
			}
			// 退化到 AppData
			_ = os.MkdirAll(filepath.Join(appArtifactsRoot, "reports"), 0o755)
			_ = os.MkdirAll(filepath.Join(appArtifactsRoot, "logs"), 0o755)
			return appArtifactsRoot, name, nil
		}
		// 自定义路径
		custom := strings.TrimSpace(cfg.FolderReportCustomPath)
		if custom == "" {
			_ = os.MkdirAll(filepath.Join(appArtifactsRoot, "reports"), 0o755)
			_ = os.MkdirAll(filepath.Join(appArtifactsRoot, "logs"), 0o755)
			return appArtifactsRoot, name, nil
		}
		if err := os.MkdirAll(filepath.Join(custom, "reports"), 0o755); err != nil {
			return "", "", fmt.Errorf("创建自定义报告目录失败: %w", err)
		}
		if err := os.MkdirAll(filepath.Join(custom, "logs"), 0o755); err != nil {
			return "", "", fmt.Errorf("创建自定义日志目录失败: %w", err)
		}
		return custom, name, nil
	}

	// zip 扫描：name 用 zip 文件名去后缀
	zipBase := filepath.Base(scanPath)
	name = strings.TrimSuffix(zipBase, filepath.Ext(zipBase))

	if cfg.ZipReportUseDefault {
		_ = os.MkdirAll(filepath.Join(appArtifactsRoot, "reports"), 0o755)
		_ = os.MkdirAll(filepath.Join(appArtifactsRoot, "logs"), 0o755)
		return appArtifactsRoot, name, nil
	}
	custom := strings.TrimSpace(cfg.ZipReportCustomPath)
	if custom == "" {
		_ = os.MkdirAll(filepath.Join(appArtifactsRoot, "reports"), 0o755)
		_ = os.MkdirAll(filepath.Join(appArtifactsRoot, "logs"), 0o755)
		return appArtifactsRoot, name, nil
	}
	if err := os.MkdirAll(filepath.Join(custom, "reports"), 0o755); err != nil {
		return "", "", fmt.Errorf("创建自定义报告目录失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(custom, "logs"), 0o755); err != nil {
		return "", "", fmt.Errorf("创建自定义日志目录失败: %w", err)
	}
	return custom, name, nil
}

// Cancel 取消任务（不等待完成，调用后立即返回）。
func (m *Manager) Cancel(taskID string) error {
	m.mu.RLock()
	t, ok := m.tasks[taskID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("任务不存在: %s", taskID)
	}
	t.Cancel()
	return nil
}

// Get 读取任务详情（按值拷贝可序列化字段；锁/日志缓冲/原子量不暴露给前端）。
// 内存里没有就从历史里捞（重启后扫描完的任务走这条路径）。
func (m *Manager) Get(taskID string) (*Task, bool) {
	m.mu.RLock()
	t, ok := m.tasks[taskID]
	m.mu.RUnlock()
	if ok {
		return m.copyTask(t), true
	}
	if m.history == nil {
		return nil, false
	}
	if h, found := m.history.Get(taskID); found {
		return taskFromHistory(h), true
	}
	return nil, false
}

// copyTask 逐字段复制，跳过 mu/cancel/cmdRunning/logs/logDropped（不可拷贝/不应序列化）。
func (m *Manager) copyTask(t *Task) *Task {
	return &Task{
		ID:          t.ID,
		Label:       t.Label,
		ProjectName: t.ProjectName,
		Path:        t.Path,
		Status:      t.Status,
		Progress:    t.Progress,
		Stage:       t.Stage,
		StartedAt:   t.StartedAt,
		FinishedAt:  t.FinishedAt,
		DurationMs:  t.DurationMs,
		ExitCode:    t.ExitCode,
		Error:       t.Error,
		ReportPath:  t.ReportPath,
		HTMLPath:    t.HTMLPath,
		LogPath:     t.LogPath,
	}
}

// Logs 从指定 offset（按行）读取日志。
// 内存里没有就从磁盘 log 文件读（历史任务走这条路径）。
func (m *Manager) Logs(taskID string, offset int) (string, error) {
	m.mu.RLock()
	t, ok := m.tasks[taskID]
	m.mu.RUnlock()
	if ok {
		all := t.LogsCopy()
		if offset < 0 || offset >= len(all) {
			offset = 0
		}
		var sb strings.Builder
		for _, e := range all[offset:] {
			sb.WriteString(e.Line)
			sb.WriteByte('\n')
		}
		return sb.String(), nil
	}
	// 退化：从 history 拿 logPath 直接读文件
	if m.history != nil {
		if h, found := m.history.Get(taskID); found && h.LogPath != "" {
			return readLogFile(h.LogPath, offset)
		}
	}
	return "", fmt.Errorf("任务不存在: %s", taskID)
}

// readLogFile 按行读取整个日志文件，丢掉前 offset 行。
func readLogFile(path string, offset int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if offset < 0 || offset >= len(lines) {
		offset = 0
	}
	return strings.Join(lines[offset:], "\n"), nil
}

// Result 读取并解析报告。
func (m *Manager) Result(taskID string) (*Report, error) {
	m.mu.RLock()
	t, ok := m.tasks[taskID]
	m.mu.RUnlock()
	if !ok {
		if m.history == nil {
			return nil, fmt.Errorf("任务不存在: %s", taskID)
		}
		h, found := m.history.Get(taskID)
		if !found || h.ReportPath == "" {
			return nil, fmt.Errorf("任务不存在: %s", taskID)
		}
		if h.Status != string(StatusSuccess) {
			return nil, fmt.Errorf("任务尚未成功完成，当前状态: %s", h.Status)
		}
		return ParseReport(taskID, h.ReportPath)
	}
	if t.Status != StatusSuccess {
		return nil, fmt.Errorf("任务尚未成功完成，当前状态: %s", t.Status)
	}
	if t.ReportPath == "" {
		return nil, fmt.Errorf("报告路径为空")
	}
	return ParseReport(taskID, t.ReportPath)
}

// ListSummaries 列出所有任务摘要（内存 + 历史，按 startedAt 倒序）。
//
// 同一个 task 优先返回内存版本（更新鲜），历史只补内存里没有的。
func (m *Manager) ListSummaries() []TaskSummary {
	m.mu.RLock()
	out := make([]TaskSummary, 0, len(m.tasks))
	for _, t := range m.tasks {
		out = append(out, t.Summary())
	}
	m.mu.RUnlock()

	if m.history != nil {
		memIDs := make(map[string]bool, len(out))
		for _, s := range out {
			memIDs[s.ID] = true
		}
		for _, h := range m.history.List() {
			if memIDs[h.ID] {
				continue
			}
			out = append(out, TaskSummary{
				ID:          h.ID,
				Label:       h.Label,
				ProjectName: h.ProjectName,
				Path:        h.Path,
				Status:      TaskStatus(h.Status),
				Progress:    h.Progress,
				StartedAt:   time.UnixMilli(h.StartedAt),
				FinishedAt:  time.UnixMilli(h.FinishedAt),
				DurationMs:  h.DurationMs,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt.After(out[j].StartedAt)
	})
	return out
}

// Delete 移除一个任务（内存 + 历史 + 项目本地历史 + 产物文件）。
func (m *Manager) Delete(taskID string) error {
	m.mu.Lock()
	t, ok := m.tasks[taskID]
	if ok {
		delete(m.tasks, taskID)
	}
	m.mu.Unlock()

	// 从历史里也拿一次（重启后历史任务不在内存）
	var reportPath, htmlPath, logPath, projectPath string
	if ok {
		reportPath = t.ReportPath
		htmlPath = t.HTMLPath
		logPath = t.LogPath
		projectPath = t.Path
	} else if m.history != nil {
		if h, found := m.history.Get(taskID); found {
			reportPath = h.ReportPath
			htmlPath = h.HTMLPath
			logPath = h.LogPath
			projectPath = h.Path
		}
	}

	if m.history != nil {
		_ = m.history.Delete(taskID)
	}

	// 同步清理项目本地历史（如果原本就走的是项目本地路径）
	if projectPath != "" && isProjectLocalPath(projectPath, reportPath) {
		_ = projectstore.DeleteEntry(projectPath, taskID)
	}

	// 清理产物文件
	if reportPath != "" {
		_ = os.Remove(reportPath)
	}
	if htmlPath != "" {
		_ = os.Remove(htmlPath)
	}
	if logPath != "" {
		_ = os.Remove(logPath)
	}
	return nil
}

// taskFromHistory 把历史 Entry 还原成 Task（用于 Get 返回给前端）。
func taskFromHistory(h history.Entry) *Task {
	return &Task{
		ID:          h.ID,
		Label:       h.Label,
		ProjectName: h.ProjectName,
		Path:        h.Path,
		Status:      TaskStatus(h.Status),
		Progress:    h.Progress,
		Stage:       h.Stage,
		StartedAt:   time.UnixMilli(h.StartedAt),
		FinishedAt:  time.UnixMilli(h.FinishedAt),
		DurationMs:  h.DurationMs,
		ExitCode:    h.ExitCode,
		Error:       h.Error,
		ReportPath:  h.ReportPath,
		HTMLPath:    h.HTMLPath,
		LogPath:     h.LogPath,
	}
}

// saveHistory 把当前 task 状态同步到历史（方便重启后 ListSummaries 看到）。
func (m *Manager) saveHistory(t *Task) {
	if m.history == nil {
		return
	}
	_ = m.history.Upsert(history.Entry{
		ID:          t.ID,
		Label:       t.Label,
		ProjectName: t.ProjectName,
		Path:        t.Path,
		Status:      string(t.Status),
		Progress:    t.Progress,
		Stage:       t.Stage,
		StartedAt:   t.StartedAt.UnixMilli(),
		FinishedAt:  t.FinishedAt.UnixMilli(),
		DurationMs:  t.DurationMs,
		ExitCode:    t.ExitCode,
		Error:       t.Error,
		ReportPath:  t.ReportPath,
		HTMLPath:    t.HTMLPath,
		LogPath:     t.LogPath,
	})
}

// saveProjectHistory 把当前 task 状态同步到项目目录的 .opensca-ui/scans.json。
// 与全局 history 是同一份 task 数据的双写（用 taskID 关联）。
//
// 如果产物路径不在项目本地（比如写入 %APPDATA% 退化了），这里跳过：
// 因为这种任务的报告并不跟项目走，没意义在项目本地留索引。
func (m *Manager) saveProjectHistory(t *Task) {
	if !isProjectLocalPath(t.Path, t.ReportPath) {
		return
	}
	_ = projectstore.UpsertEntry(t.Path, projectstore.Entry{
		ID:          t.ID,
		Label:       t.Label,
		ProjectName: t.ProjectName,
		Path:        t.Path,
		Status:      string(t.Status),
		Progress:    t.Progress,
		Stage:       t.Stage,
		StartedAt:   t.StartedAt.UnixMilli(),
		FinishedAt:  t.FinishedAt.UnixMilli(),
		DurationMs:  t.DurationMs,
		ExitCode:    t.ExitCode,
		Error:       t.Error,
		ReportPath:  t.ReportPath,
		HTMLPath:    t.HTMLPath,
		LogPath:     t.LogPath,
	})
}

// isProjectLocalPath 报告路径是否在项目本地的 `.opensca-ui/` 里。
//
// 布局：
//   - <project>/.opensca-ui/reports/<basename>.{json,html}
//   - <project>/.opensca-ui/logs/<taskID>.log
func isProjectLocalPath(projectPath, reportPath string) bool {
	if projectPath == "" || reportPath == "" {
		return false
	}
	rel, err := filepath.Rel(projectPath, reportPath)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	if filepath.HasPrefix(rel, ".opensca-ui/reports/") {
		return true
	}
	if filepath.HasPrefix(rel, ".opensca-ui/logs/") {
		return true
	}
	return false
}

// failTask 立即把任务置为 Failed。
func (m *Manager) failTask(t *Task, err error) {
	t.MarkDone(StatusFailed, -1, err.Error(), "")
	m.emit("scan:update", map[string]any{"taskID": t.ID, "status": string(StatusFailed)})
	m.emit("scan:done", map[string]any{
		"taskID":     t.ID,
		"status":     string(StatusFailed),
		"durationMs": t.DurationMs,
		"reportPath": "",
	})
}

// runOne worker 调起一个任务。包含：构建参数、启动 CLI、行回调、终态判定。
func (m *Manager) runOne(t *Task) {
	cfg := m.cfg.Get()

	// 设置取消函数
	ctx, cancel := context.WithCancel(m.rootCtx)
	t.SetCancel(func() {
		cancel()
		// Windows 上需要手动 taskkill（exec.CommandContext 已经会处理，但保险起见再触发一次）
		if t.cmdRunning.Load() {
			killProcessGroup(-1) // pgid 在 runCLI 内部已处理
		}
	})

	t.MarkRunning()
	m.emit("scan:update", map[string]any{"taskID": t.ID, "status": string(StatusRunning)})
	m.emit("scan:started", map[string]any{"taskID": t.ID, "startedAt": t.StartedAt.UnixMilli()})

	// 行回调：写内存日志、推事件、解析进度
	onLine := func(line string) {
		t.AppendLog(line)
		m.emit("scan:log", map[string]any{"taskID": t.ID, "line": line, "ts": time.Now().UnixMilli()})
		if pct, stage, ok := parseProgress(line); ok {
			t.Progress = pct
			t.Stage = stage
			m.emit("scan:progress", map[string]any{"taskID": t.ID, "percent": pct, "stage": stage})
		}
	}

	// 构造 CLI 参数（一次扫描同时输出 JSON + HTML）
	args := buildArgs(t.Path, t.ReportPath, t.HTMLPath, cfg.Token, cfg.LocalDB)

	// 工作目录设为 CLI 所在目录：opensca-cli v3.x 会从运行目录自动读取
	// config.json（内置 CLI 随包带了 config.json + db-demo.json，
	// 自选的 CLI 也能带上它自己目录里的 config），缺了会报
	// "not config vuln database origin"。
	workDir := ""
	if cfg.CLIPath != "" {
		workDir = filepath.Dir(cfg.CLIPath)
	}

	res := runCLI(ctx, CLIRequest{
		CLIPath: cfg.CLIPath,
		Args:    args,
		WorkDir: workDir,
		Env:     nil,
		Timeout: 30 * time.Minute,
	}, onLine)

	// 终态判定
	switch {
	case ctx.Err() == context.Canceled && t.Status != StatusFailed:
		t.MarkDone(StatusCanceled, res.ExitCode, "用户取消", "")
	case res.Err != nil && ctx.Err() == context.DeadlineExceeded:
		t.MarkDone(StatusFailed, res.ExitCode, "扫描超时（30 分钟）", "")
	case res.ExitCode == 0:
		// 检查报告是否生成
		if _, err := os.Stat(t.ReportPath); err != nil {
			t.MarkDone(StatusFailed, res.ExitCode, "CLI 退出 0 但未生成报告", "")
		} else {
			t.MarkDone(StatusSuccess, res.ExitCode, "", t.ReportPath)
			// 把日志落盘
			m.flushLog(t)
		}
	default:
		t.MarkDone(StatusFailed, res.ExitCode, fmt.Sprintf("CLI 退出码 %d", res.ExitCode), "")
		m.flushLog(t)
	}

	m.emit("scan:update", map[string]any{"taskID": t.ID, "status": string(t.Status)})
	m.emit("scan:done", map[string]any{
		"taskID":     t.ID,
		"status":     string(t.Status),
		"durationMs": t.DurationMs,
		"reportPath": t.ReportPath,
	})
	// 终态写入历史（这样重启后 ListSummaries 仍能看到这条任务）
	m.saveHistory(t)
	// 同时写到项目本地 .opensca-ui/scans.json（如果产物本来就落在项目里）
	m.saveProjectHistory(t)
}

// flushLog 把任务完整日志写入磁盘文件。
func (m *Manager) flushLog(t *Task) {
	if t.LogPath == "" {
		return
	}
	entries := t.LogsCopy()
	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString(e.Line)
		sb.WriteByte('\n')
	}
	_ = os.WriteFile(t.LogPath, []byte(sb.String()), 0o644)
}

// newID 生成 8 字节随机 hex 字符串作为 taskID。
func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// deriveProjectName 从扫描路径推导项目名：
//   - 目录 → basename
//   - 文件（含 zip/tar.gz 等压缩包）→ basename 去扩展名（兼容 .tar.gz 双扩展）
//
// 推导失败或空路径时返回空串，调用方决定兜底。
// StartRequest 是 Submit 的入参结构。
type StartRequest struct {
	Path  string
	Label string
}