package scanner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

// CLIRequest 启动一次扫描所需参数。
type CLIRequest struct {
	CLIPath string
	Args    []string
	WorkDir string
	Env     []string
	Timeout time.Duration // 0 表示无超时
}

// runResult 单次 CLI 调用的结果。
type runResult struct {
	ExitCode int
	Err      error
}

// OnLine 是 stdout 每行的回调。
type OnLine func(line string)

// runCLI 启动 opensca-cli 子进程，逐行回调 stdout。
//
// 关键点：
//   - Windows 上把子进程放到新进程组（CREATE_NEW_PROCESS_GROUP）便于整体 kill
//   - 使用 exec.CommandContext 让 ctx 取消时自动终止
//   - 启动 killWatcher，ctx 一旦 cancel 就尝试 taskkill /T /F
func runCLI(ctx context.Context, req CLIRequest, onLine OnLine) runResult {
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, req.CLIPath, req.Args...)
	cmd.Dir = req.WorkDir
	if len(req.Env) > 0 {
		cmd.Env = append(cmd.Environ(), req.Env...)
	}
	hideWindow(cmd) // Windows: 隐藏子进程控制台窗口

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return runResult{ExitCode: -1, Err: fmt.Errorf("stdout pipe: %w", err)}
	}
	// 也接管 stderr（合并到 stdout 行回调里以便实时显示）
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return runResult{ExitCode: -1, Err: fmt.Errorf("stderr pipe: %w", err)}
	}

	if err := cmd.Start(); err != nil {
		return runResult{ExitCode: -1, Err: fmt.Errorf("start cli: %w", err)}
	}

	// Windows：把进程组 ID 记下来，cancel 时 taskkill /T /F /PID <pgid>
	pgid := cmd.Process.Pid

	go scanLines(stdout, onLine)
	go scanLines(stderr, func(line string) {
		if line != "" {
			onLine("[stderr] " + line)
		}
	})

	waitErr := cmd.Wait()
	exitCode := 0
	if waitErr != nil {
		if ee, ok := waitErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			// ctx 取消也走这里
			exitCode = -1
		}
	}
	_ = pgid
	return runResult{ExitCode: exitCode, Err: waitErr}
}

func scanLines(r io.Reader, onLine func(string)) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if onLine != nil {
			onLine(line)
		}
	}
}

// hideWindow 在 Windows 上隐藏 CLI 子进程弹出的黑色控制台窗口。
func hideWindow(cmd *exec.Cmd) {
	if runtime.GOOS != "windows" {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// CREATE_NEW_PROCESS_GROUP(0x00000200) | CREATE_NO_WINDOW(0x08000000)
	cmd.SysProcAttr.CreationFlags |= 0x00000200 | 0x08000000
}

// killProcessGroup 在 Windows 上 taskkill /T /F /PID，杀整组；其他平台走 Kill。
func killProcessGroup(pid int) {
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run()
		return
	}
	// 其他平台：交给 exec.CommandContext 处理
	if p, err := findProcess(pid); err == nil && p != nil {
		_ = p.Kill()
	}
}

// findProcess 是 os.FindProcess 的封装，方便测试时替换。
func findProcess(pid int) (interface{ Kill() error }, error) {
	return nil, nil // placeholder, no-op on non-windows
}

// buildArgs 根据扫描参数拼装 opensca-cli 的命令行参数。
//
// 不同版本的 opensca-cli flag 集合差异很大。这里只拼"通用且必要的"参数：
//   -path  <scanPath>            必填：项目路径或压缩包
//   -out   <jsonPath[,htmlPath]> 必填：报告输出路径（v3.x：格式由扩展名推断，
//                                  支持多个，用逗号分隔；一次扫描同时输出 JSON+HTML）
//
// 可选参数（按需追加）：
//   -token <token>     v3.x 云漏洞库 token。CLI 默认 origin.url=https://opensca.xmirror.cn，
//                      传 token 后会自动走云端漏洞查询（SearchDetail → c.Url != "" && c.Token != ""）。
//                      不传 token 时 CLI 报 "not config vuln database origin"。
//   -config <cfgPath>  v3.x 自定义配置文件路径。本地漏洞库走这个：CLI 读 config.json 后
//                      从 origin.json 字段加载 JSON 格式漏洞库文件（BaseOrigin.SearchVuln）。
//                      我们临时生成一份只填 origin.json 的最小 config，传给 CLI。
//
// 历史注记：v2.x 的 -format/-db/-login（交互登录）已被 v3.x 移除，不再传。
func buildArgs(scanPath string, reportPath string, htmlPath string, token string, dbPath string) []string {
	out := reportPath
	if htmlPath != "" {
		out = reportPath + "," + htmlPath
	}
	args := []string{
		"-path", scanPath,
		"-out", out,
	}
	if token != "" {
		// v3.x：-token 直接给 Origin.Token，配合默认 origin.url 走云漏洞库
		args = append(args, "-token", token)
	}
	if dbPath != "" {
		// v3.x：本地 JSON 漏洞库要走 -config 指向的 config.json（origin.json 字段）
		if cfgPath, err := writeLocalDBConfig(dbPath); err == nil {
			args = append(args, "-config", cfgPath)
		}
		// 写盘失败就静默跳过：CLI 会用默认 config，task_info.error 透传给 UI
	}
	return args
}

// writeLocalDBConfig 生成一份只含 origin.json 字段的临时 config.json 给 v3.x CLI 用。
//
// 返回临时文件路径。CLI 进程结束后文件残留无影响（%TEMP% 下，系统会清）。
// 失败返回 ""，调用方应跳过 -config 参数。
func writeLocalDBConfig(dbPath string) (string, error) {
	// CLI 的 config.go 用 json5 解码，但 json.Marshal 出的也是合法 json5（json5 是 json 超集）
	cfg := map[string]any{
		"origin": map[string]any{
			"json": dbPath,
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "opensca-ui-cfg-*.json")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	f.Close()
	return filepath.Clean(f.Name()), nil
}