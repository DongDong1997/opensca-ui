package scanner

import (
	"regexp"
	"strconv"
	"strings"
)

// progressLineRegex 匹配 opensca-cli 类似 "[INFO] 12/100 (12%) xxx" 的输出。
// 也兼容 "12%" / "12 %" / "progress: 12" 等变体。
var progressPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(\d{1,3})\s*%`),
	regexp.MustCompile(`(?i)progress[:\s]+(\d{1,3})`),
	regexp.MustCompile(`(?i)\[(\d{1,3})/(\d{1,3})\]`),
	regexp.MustCompile(`(?i)(\d{1,3})/(\d{1,3})\s*\(\s*(\d{1,3})\s*%\s*\)`),
}

// parseProgress 试图从单行日志中提取 (0-100, stage)。
// 失败时返回 (currentProgress, "", false)。
func parseProgress(line string) (int, string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return 0, "", false
	}
	for _, re := range progressPatterns {
		m := re.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		switch len(m) {
		case 2:
			if n, err := strconv.Atoi(m[1]); err == nil && n >= 0 && n <= 100 {
				return n, stageFromLine(line), true
			}
		case 3:
			// "12/100"
			if cur, err1 := strconv.Atoi(m[1]); err1 == nil {
				if total, err2 := strconv.Atoi(m[2]); err2 == nil && total > 0 {
					p := cur * 100 / total
					if p > 100 {
						p = 100
					}
					return p, stageFromLine(line), true
				}
			}
		case 4:
			// "12/100 (12%)"
			if n, err := strconv.Atoi(m[3]); err == nil && n >= 0 && n <= 100 {
				return n, stageFromLine(line), true
			}
		}
	}
	return 0, "", false
}

// stageFromLine 从日志中提取阶段描述（启发式）。
func stageFromLine(line string) string {
	// 去掉前缀的 "[INFO] [WARN] ..." 之类
	if idx := strings.Index(line, "]"); idx > 0 && idx < 16 {
		line = strings.TrimSpace(line[idx+1:])
	}
	if len(line) > 80 {
		return line[:80] + "..."
	}
	return line
}