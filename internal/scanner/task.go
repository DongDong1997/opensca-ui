package scanner

import (
	"sync"
	"sync/atomic"
	"time"
)

// TaskStatus 任务状态机。
type TaskStatus string

const (
	StatusPending  TaskStatus = "pending"
	StatusRunning  TaskStatus = "running"
	StatusSuccess  TaskStatus = "success"
	StatusFailed   TaskStatus = "failed"
	StatusCanceled TaskStatus = "canceled"
)

// LogEntry 单行日志。
type LogEntry struct {
	TS   time.Time `json:"ts"`
	Line string    `json:"line"`
}

// Task 一个扫描任务。
type Task struct {
	ID          string     `json:"id"`
	Label       string     `json:"label"`       // 任务备注（用户自由填）
	ProjectName string     `json:"projectName"` // 项目名：取自文件夹名（或 zip 去后缀），作为记录的绑定键
	Path        string     `json:"path"`
	Status      TaskStatus `json:"status"`
	Progress    int        `json:"progress"`
	Stage       string     `json:"stage"`
	StartedAt   time.Time  `json:"startedAt"`
	FinishedAt  time.Time  `json:"finishedAt"`
	DurationMs  int64      `json:"durationMs"`
	ExitCode    int        `json:"exitCode"`
	Error       string     `json:"error"`
	ReportPath  string     `json:"reportPath"` // JSON 报告，UI 解析用
	HTMLPath    string     `json:"htmlPath"`   // HTML 报告，给用户看的
	LogPath     string     `json:"logPath"`

	mu         sync.Mutex
	cancel     func()       // context.CancelFunc
	cmdRunning atomic.Bool  // 防止重复 kill
	logs       []LogEntry   // 内存中的日志缓冲（落盘前）
	logDropped atomic.Int64 // 环形丢弃的日志条数（避免前端/内存爆）
}

// SetCancel 设置取消函数。
func (t *Task) SetCancel(fn func()) { t.cancel = fn }

// Cancel 触发取消（幂等）。
func (t *Task) Cancel() {
	t.mu.Lock()
	fn := t.cancel
	cmdRunning := t.cmdRunning.Load()
	t.mu.Unlock()
	if cmdRunning && fn != nil {
		fn()
	}
}

// AppendLog 追加一行日志。
func (t *Task) AppendLog(line string) {
	t.mu.Lock()
	t.logs = append(t.logs, LogEntry{TS: time.Now(), Line: line})
	t.mu.Unlock()
}

// LogsCopy 返回当前内存日志的拷贝。
func (t *Task) LogsCopy() []LogEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]LogEntry, len(t.logs))
	copy(out, t.logs)
	return out
}

// MarkRunning 进入 Running 状态。
func (t *Task) MarkRunning() {
	t.Status = StatusRunning
	t.StartedAt = time.Now()
	t.Stage = "扫描中"
	t.cmdRunning.Store(true)
}

// MarkDone 终态。
func (t *Task) MarkDone(status TaskStatus, exitCode int, errMsg string, reportPath string) {
	t.Status = status
	t.ExitCode = exitCode
	t.Error = errMsg
	t.FinishedAt = time.Now()
	t.DurationMs = t.FinishedAt.Sub(t.StartedAt).Milliseconds()
	if t.Progress < 100 && status == StatusSuccess {
		t.Progress = 100
	}
	if reportPath != "" {
		t.ReportPath = reportPath
	}
	t.Stage = statusLabel(status)
	t.cmdRunning.Store(false)
}

func statusLabel(s TaskStatus) string {
	switch s {
	case StatusPending:
		return "等待中"
	case StatusRunning:
		return "扫描中"
	case StatusSuccess:
		return "完成"
	case StatusFailed:
		return "失败"
	case StatusCanceled:
		return "已取消"
	}
	return string(s)
}

// TaskSummary 给前端列表用，比 Task 字段更少。
type TaskSummary struct {
	ID          string     `json:"id"`
	Label       string     `json:"label"`
	ProjectName string     `json:"projectName"`
	Path        string     `json:"path"`
	Status      TaskStatus `json:"status"`
	Progress    int        `json:"progress"`
	StartedAt   time.Time  `json:"startedAt"`
	FinishedAt  time.Time  `json:"finishedAt"`
	DurationMs  int64      `json:"durationMs"`
}

// Summary 生成摘要。
func (t *Task) Summary() TaskSummary {
	return TaskSummary{
		ID:          t.ID,
		Label:       t.Label,
		ProjectName: t.ProjectName,
		Path:        t.Path,
		Status:      t.Status,
		Progress:    t.Progress,
		StartedAt:   t.StartedAt,
		FinishedAt:  t.FinishedAt,
		DurationMs:  t.DurationMs,
	}
}