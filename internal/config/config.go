package config

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"

	"opensca-ui/internal/platform"
)

// Config 持久化在 %APPDATA%/opensca-ui/config.json。
//
// 字段顺序保持稳定：JSON tag 是契约，新增字段用 omitempty + 默认值。
type Config struct {
	CLIPath       string `json:"cliPath"`
	CLIValid      bool   `json:"cliValid"`
	CLIVersion    string `json:"cliVersion"`
	// CliPathManual = true 表示用户在设置/欢迎页手动配置过 CLI 路径。
	// 为 false（默认）时，启动会自动用内置 CLI（安装路径下的 opensca-cli）作为默认。
	// 用于区分"用户主动选择"和"默认值/旧版遗留配置"，避免每次启动都覆盖用户的设置。
	CliPathManual bool   `json:"cliPathManual,omitempty"`
	MaxConcurrent int    `json:"maxConcurrent"`
	Token         string `json:"token"`
	LocalDB       string `json:"localDB"`
	Theme         string `json:"theme"` // "light" | "dark"

	// FolderReportUseDefault = true 时：文件夹扫描报告落项目本地 .opensca-ui/reports，
	//                            不可写时回退 %APPDATA%/opensca-ui/reports。
	// FolderReportUseDefault = false 时：报告落 FolderReportCustomPath。
	FolderReportUseDefault bool   `json:"folderReportUseDefault"`
	FolderReportCustomPath string `json:"folderReportCustomPath"`

	// ZipReportUseDefault = true 时：压缩包扫描报告落 %APPDATA%/opensca-ui/reports。
	// ZipReportUseDefault = false 时：报告落 ZipReportCustomPath。
	ZipReportUseDefault bool   `json:"zipReportUseDefault"`
	ZipReportCustomPath string `json:"zipReportCustomPath"`
}

// CliInfo 是 CheckCli 给前端返回的结构。
type CliInfo struct {
	Path      string `json:"path"`
	Valid     bool   `json:"valid"`
	Version   string `json:"version"`
	Message   string `json:"message"`
	RawOutput string `json:"rawOutput,omitempty"` // CLI 的原始输出，便于诊断
}

// 默认值
const DefaultMaxConcurrent = 3

func Default() Config {
	return Config{
		CLIPath:               "",
		CLIValid:              false,
		CLIVersion:            "",
		MaxConcurrent:         DefaultMaxConcurrent,
		Token:                 "",
		LocalDB:               "",
		Theme:                 "light",
		FolderReportUseDefault: true,
		FolderReportCustomPath: "",
		ZipReportUseDefault:    true,
		ZipReportCustomPath:    "",
	}
}

// Store 是带文件落盘的配置管理器，进程内单例。
type Store struct {
	mu   sync.RWMutex
	cfg  Config
	path string // config.json 完整路径，Open 后必填
}

// Open 加载配置（首次启动时文件不存在用默认值），返回的 Store 一定能写盘。
func Open() (*Store, error) {
	root, err := platform.AppDataDir()
	if err != nil {
		return nil, err
	}
	s := &Store{
		cfg:  Default(),
		path: filepath.Join(root, "config.json"),
	}
	if err := s.load(); err != nil {
		// JSON 损坏时仍要返回能写盘的 Store（不要 fallback 到 path="" 的空 Store，
		// 否则后续所有写入都会静默失败 → 用户配置看似保存了但下次启动全丢）。
		log.Printf("config load failed (%v): 使用默认值覆盖原文件", err)
		_ = s.flush()
	}
	return s, nil
}

// Path 返回配置文件的绝对路径（暴露给前端用于诊断）。
func (s *Store) Path() string {
	return s.path
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // 第一次启动，配置文件不存在是预期
		}
		return err
	}
	// 合并默认值：避免磁盘缺字段导致运行异常
	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
	return nil
}

// Get 浅拷贝返回，避免调用方修改内部状态
func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// Set 全量替换（小心使用，优先用下面的字段级 setter）
func (s *Store) Set(c Config) error {
	s.mu.Lock()
	s.cfg = c
	s.mu.Unlock()
	return s.flush()
}

// SetCLIPath 同时清空 CLIValid/CLIVersion（等下次 CheckCli 重新填充）。
// 不清 CliPathManual 标记 —— 供启动时写入"内置 CLI 默认路径"使用。
func (s *Store) SetCLIPath(p string) error {
	s.mu.Lock()
	s.cfg.CLIPath = p
	s.cfg.CLIValid = false
	s.cfg.CLIVersion = ""
	s.mu.Unlock()
	return s.flush()
}

// SetCLIPathManual 用户在 UI 手动配置 CLI 路径时调用。
// 与 SetCLIPath 的区别：额外把 CliPathManual 置 true，
// 这样后续启动 ensureBundledCli 不会再覆盖用户的主动选择。
func (s *Store) SetCLIPathManual(p string) error {
	s.mu.Lock()
	s.cfg.CLIPath = p
	s.cfg.CLIValid = false
	s.cfg.CLIVersion = ""
	s.cfg.CliPathManual = true
	s.mu.Unlock()
	return s.flush()
}

func (s *Store) SetCLIStatus(valid bool, version string) error {
	s.mu.Lock()
	s.cfg.CLIValid = valid
	s.cfg.CLIVersion = version
	s.mu.Unlock()
	return s.flush()
}

func (s *Store) SetMaxConcurrent(n int) error {
	if n < 1 {
		n = 1
	}
	if n > 10 {
		n = 10
	}
	s.mu.Lock()
	s.cfg.MaxConcurrent = n
	s.mu.Unlock()
	return s.flush()
}

func (s *Store) SetToken(t string) error {
	s.mu.Lock()
	s.cfg.Token = t
	s.mu.Unlock()
	return s.flush()
}

func (s *Store) SetLocalDB(p string) error {
	s.mu.Lock()
	s.cfg.LocalDB = p
	s.mu.Unlock()
	return s.flush()
}

func (s *Store) SetTheme(t string) error {
	if t != "light" && t != "dark" {
		t = "light"
	}
	s.mu.Lock()
	s.cfg.Theme = t
	s.mu.Unlock()
	return s.flush()
}

// SetFolderReportLocation 一次设置文件夹扫描报告的两个相关字段。
func (s *Store) SetFolderReportLocation(useDefault bool, customPath string) error {
	s.mu.Lock()
	s.cfg.FolderReportUseDefault = useDefault
	s.cfg.FolderReportCustomPath = customPath
	s.mu.Unlock()
	return s.flush()
}

// SetZipReportLocation 一次设置压缩包扫描报告的两个相关字段。
func (s *Store) SetZipReportLocation(useDefault bool, customPath string) error {
	s.mu.Lock()
	s.cfg.ZipReportUseDefault = useDefault
	s.cfg.ZipReportCustomPath = customPath
	s.mu.Unlock()
	return s.flush()
}

// flush 同步、原子地写盘：先写 .tmp 再 rename，避免半写状态。
//
// 改用同步写的原因：之前的 500ms debounce 在用户快速关闭应用时 timer 没机会触发，
// 哪怕 OnShutdown 调用了 FlushNow，也可能在并发路径上丢数据。配置写入频率很低，
// 同步写完全够用，并且"任何一次写入返回 err 就一定没落盘"——便于诊断。
func (s *Store) flush() error {
	if s.path == "" {
		return errors.New("config: 路径未初始化")
	}
	data, err := json.MarshalIndent(s.Get(), "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// FlushNow 兼容旧接口，现在与 flush 等价（同步写）。
func (s *Store) FlushNow() error {
	return s.flush()
}
