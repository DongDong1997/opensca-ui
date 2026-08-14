package recent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"opensca-ui/internal/platform"
	"opensca-ui/internal/projectname"
)

// MaxEntries 限制最近项目数量；新条目超过上限时按 LastAt 尾部淘汰。
const MaxEntries = 20

// Entry 描述一个最近打开的项目。
//
// 字段顺序保持稳定：JSON tag 是契约。
type Entry struct {
	Path     string `json:"path"`
	Label    string `json:"label"`
	// ProjectName 由 path 推导的稳定项目名（目录取 basename、压缩包去扩展名）。
	// 用作显示名 + 在 history 页面里做"按项目分组"的绑定键。
	// 与 Label 的区别：Label 是用户每次扫描的自由备注，会变；ProjectName 是
	// 从路径算出来的，同一目录/压缩包永远一致。
	ProjectName string `json:"projectName"`
	LastAt      int64  `json:"lastAt"`   // UnixMilli
	UseCount    int    `json:"useCount"` // 累计扫描次数
}

// Store 持久化最近项目列表（保存在 %APPDATA%/opensca-ui/recent.json）。
type Store struct {
	mu      sync.RWMutex
	path    string
	entries []Entry
}

// Open 加载最近项目（文件不存在则用空列表）。
func Open() (*Store, error) {
	root, err := platform.AppDataDir()
	if err != nil {
		return nil, err
	}
	s := &Store{
		path:    filepath.Join(root, "recent.json"),
		entries: []Entry{},
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
	// 过滤空路径和空时间戳，避免脏数据
	clean := make([]Entry, 0, len(list))
	for _, e := range list {
		if e.Path == "" {
			continue
		}
		if e.LastAt <= 0 {
			continue
		}
		clean = append(clean, e)
	}
	s.mu.Lock()
	s.entries = clean
	s.mu.Unlock()
	return nil
}

// List 浅拷贝返回（前端按顺序展示，无需排序）。
func (s *Store) List() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

// Add 记录一次访问。若 path 已存在则只更新 LastAt/UseCount/ProjectName；否则追加。
//
//   - label 由调用方（扫描入口）传入；为空时回退到 path 的 basename。
//   - projectName 始终由 path 重新推导（与 Label 无关），保证显示名稳定。
func (s *Store) Add(path, label string) error {
	if path == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if label == "" {
		label = filepath.Base(path)
	}
	projectName := projectname.Derive(path)
	now := time.Now().UnixMilli()

	found := false
	for i := range s.entries {
		if s.entries[i].Path == path {
			s.entries[i].LastAt = now
			s.entries[i].UseCount++
			if label != "" {
				s.entries[i].Label = label
			}
			if projectName != "" {
				s.entries[i].ProjectName = projectName
			}
			found = true
		}
	}
	if !found {
		s.entries = append(s.entries, Entry{
			Path:        path,
			Label:       label,
			ProjectName: projectName,
			LastAt:      now,
			UseCount:    1,
		})
	}

	// 按 LastAt 倒序排
	sort.SliceStable(s.entries, func(i, j int) bool {
		return s.entries[i].LastAt > s.entries[j].LastAt
	})

	// 截断到上限
	if len(s.entries) > MaxEntries {
		s.entries = s.entries[:MaxEntries]
	}

	return s.flush()
}

// Remove 按路径删除一条。
func (s *Store) Remove(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entries {
		if s.entries[i].Path == path {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			return s.flush()
		}
	}
	return nil
}

// Clear 清空所有记录。
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = []Entry{}
	return s.flush()
}

// flush 同步写盘（与 config.Store 保持一致：synchronous atomic write）。
func (s *Store) flush() error {
	if s.path == "" {
		return errors.New("recent: 路径未初始化")
	}
	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
