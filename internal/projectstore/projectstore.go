package projectstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// DirName 是项目目录下用于存扫描元数据的隐藏子目录名。
//
// 用途：scans.json（任务索引）+ logs/（执行日志）。报告产物本身放在
// 可见的 `reports/` 目录，不在这里。
const DirName = ".opensca-ui"

// Entry 是项目本地的扫描记录（与 history.Entry 字段一致，但不持久化所有 Task 字段）。
//
// 字段顺序稳定：JSON tag 是契约。
type Entry struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	ProjectName string `json:"projectName"`
	Path        string `json:"path"`
	Status      string `json:"status"`
	Progress    int    `json:"progress"`
	Stage       string `json:"stage"`
	StartedAt   int64  `json:"startedAt"`  // UnixMilli
	FinishedAt  int64  `json:"finishedAt"` // UnixMilli
	DurationMs  int64  `json:"durationMs"`
	ExitCode    int    `json:"exitCode"`
	Error       string `json:"error"`
	ReportPath  string `json:"reportPath"`
	HTMLPath    string `json:"htmlPath"`
	LogPath     string `json:"logPath"`
}

// ScanPath 是项目根目录的元数据子目录路径（`.opensca-ui/`）。
func ScanPath(projectRoot string) string {
	return filepath.Join(projectRoot, DirName)
}

// ReportsDir 返回报告目录：`<projectRoot>/.opensca-ui/reports`。
//
// 报告（JSON + HTML）放在 `.opensca-ui/` 内部，与 scans.json / logs 一起归拢，
// 避免在项目根目录散落多个 opensca-ui 相关子目录。
func ReportsDir(projectRoot string) string {
	return filepath.Join(projectRoot, DirName, "reports")
}

// LogPath 给定 taskID 和项目根目录，返回执行日志路径（留在 `.opensca-ui/logs/`）。
func LogPath(projectRoot, taskID string) string {
	return filepath.Join(projectRoot, DirName, "logs", taskID+".log")
}

// HistoryPath 项目本地扫描索引文件路径（`.opensca-ui/scans.json`）。
func HistoryPath(projectRoot string) string {
	return filepath.Join(projectRoot, DirName, "scans.json")
}

// MakeReportPaths 在指定目录下生成新的报告文件路径（jsonPath, htmlPath）。
//
// 文件名格式：`<name>_<YYYYMMDD>_<HHMMSS>[_<seq>].{json,html}`
//
// 同一秒内多次扫描会按 `_2`、`_3`... 递增后缀避免冲突。JSON 与 HTML 共享同一基础名，
// 保证任务报告在文件管理器里成对出现。
//
// 参数：
//   - dir:  报告所在目录（调用方保证已创建或可创建）
//   - name: 文件名主体（通常是项目文件夹名或 zip 文件去后缀名）
//   - when: 扫描起始时间，用于文件名中的时间戳
func MakeReportPaths(dir, name string, when time.Time) (jsonPath, htmlPath string, err error) {
	if dir == "" {
		return "", "", errors.New("projectstore: 目录为空")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("创建报告目录失败: %w", err)
	}

	if name == "" || name == "." || name == string(filepath.Separator) {
		name = "scan"
	}
	stamp := when.Format("20060102_150405")
	base := fmt.Sprintf("%s_%s", name, stamp)

	// 检查冲突：基础名 / _2 / _3 ...
	for i := 1; i < 1000; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s_%d", base, i)
		}
		jsonCandidate := filepath.Join(dir, candidate+".json")
		htmlCandidate := filepath.Join(dir, candidate+".html")
		if fileExists(jsonCandidate) || fileExists(htmlCandidate) {
			continue
		}
		return jsonCandidate, htmlCandidate, nil
	}
	return "", "", errors.New("projectstore: 报告文件名冲突次数过多")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Ensure 在项目目录下创建必要的子目录：
//
//   - .opensca-ui/             元数据根
//   - .opensca-ui/reports/     报告目录（JSON + HTML）
//   - .opensca-ui/logs/        执行日志目录
//
// 项目目录本身若不存在也一并创建。
func Ensure(projectRoot string) error {
	if projectRoot == "" {
		return errors.New("projectstore: 空路径")
	}
	dirs := []string{
		projectRoot,
		filepath.Join(projectRoot, DirName),
		filepath.Join(projectRoot, DirName, "reports"),
		filepath.Join(projectRoot, DirName, "logs"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// IsUsable 判断项目目录是否可写入。测试在 `.opensca-ui/reports/` 上做，
// 因为报告是主要产物。
func IsUsable(projectRoot string) bool {
	if projectRoot == "" {
		return false
	}
	if _, err := os.Stat(projectRoot); err != nil {
		return false
	}
	if err := Ensure(projectRoot); err != nil {
		return false
	}
	test := filepath.Join(ReportsDir(projectRoot), ".write-test")
	if err := os.WriteFile(test, []byte("ok"), 0o644); err != nil {
		return false
	}
	_ = os.Remove(test)
	return true
}

// LoadHistory 加载项目本地扫描索引（文件不存在则空）。
func LoadHistory(projectRoot string) ([]Entry, error) {
	return loadHistoryAt(HistoryPath(projectRoot))
}

func loadHistoryAt(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var list []Entry
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	clean := make([]Entry, 0, len(list))
	for _, e := range list {
		if e.ID == "" {
			continue
		}
		clean = append(clean, e)
	}
	return clean, nil
}

// SaveHistory 覆盖写项目本地扫描索引（按 StartedAt 倒序）。
func SaveHistory(projectRoot string, entries []Entry) error {
	return saveHistoryAt(HistoryPath(projectRoot), entries)
}

// UpsertEntry 在项目本地扫描索引里插入或更新一条（按 ID 匹配）。
func UpsertEntry(projectRoot string, e Entry) error {
	if e.ID == "" {
		return nil
	}
	list, err := loadHistoryAt(HistoryPath(projectRoot))
	if err != nil {
		return err
	}
	found := false
	for i := range list {
		if list[i].ID == e.ID {
			list[i] = e
			found = true
			break
		}
	}
	if !found {
		list = append(list, e)
	}
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].StartedAt > list[j].StartedAt
	})
	// 截断到 100 条
	if len(list) > 100 {
		list = list[:100]
	}
	return saveHistoryAt(HistoryPath(projectRoot), list)
}

// DeleteEntry 从项目本地扫描索引中删除一条。
func DeleteEntry(projectRoot, id string) error {
	list, err := loadHistoryAt(HistoryPath(projectRoot))
	if err != nil {
		return err
	}
	out := make([]Entry, 0, len(list))
	for _, e := range list {
		if e.ID != id {
			out = append(out, e)
		}
	}
	return saveHistoryAt(HistoryPath(projectRoot), out)
}

// fsLock 防止同一项目下并发写扫描记录的竞态（针对多 worker 同时写不同 task）。
var fsLock sync.Mutex

func saveHistoryAt(path string, entries []Entry) error {
	fsLock.Lock()
	defer fsLock.Unlock()
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}