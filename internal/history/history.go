package history

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"opensca-ui/internal/platform"
)

// MaxEntries 任务历史最多保留多少条；超出按 StartedAt 升序淘汰旧记录。
const MaxEntries = 100

// Entry 是任务在历史里的精简记录（只保存 Summary 字段，足够列表展示）。
//
// 字段顺序保持稳定：JSON tag 是契约。
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

// Store 持久化任务历史（保存在 %APPDATA%/opensca-ui/history.json）。
type Store struct {
	mu      sync.RWMutex
	path    string
	entries map[string]Entry
}

// Open 加载历史（文件不存在则用空 map）。
func Open() (*Store, error) {
	root, err := platform.AppDataDir()
	if err != nil {
		return nil, err
	}
	s := &Store{
		path:    filepath.Join(root, "history.json"),
		entries: make(map[string]Entry),
	}
	_ = s.load()
	return s, nil
}

// Path 返回磁盘文件位置，仅用于前端诊断。
func (s *Store) Path() string {
	return s.path
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var list []Entry
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	s.mu.Lock()
	for _, e := range list {
		if e.ID == "" {
			continue
		}
		s.entries[e.ID] = e
	}
	s.mu.Unlock()
	return nil
}

// List 按 StartedAt 倒序返回所有条目。
func (s *Store) List() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt > out[j].StartedAt
	})
	return out
}

// Get 按 ID 读取一条；不存在返回 false。
func (s *Store) Get(id string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[id]
	return e, ok
}

// Upsert 写入或更新一条（终态时调）。
func (s *Store) Upsert(e Entry) error {
	if e.ID == "" {
		return nil
	}
	s.mu.Lock()
	s.entries[e.ID] = e
	s.mu.Unlock()
	return s.flush()
}

// Delete 按 ID 删除。
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	delete(s.entries, id)
	s.mu.Unlock()
	return s.flush()
}

// flush 同步写盘（与 config.Store / recent.Store 保持一致的同步原子写）。
func (s *Store) flush() error {
	if s.path == "" {
		return errors.New("history: 路径未初始化")
	}
	data, err := json.MarshalIndent(s.List(), "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
