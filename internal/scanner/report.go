package scanner

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// Vuln 与前端 api/types.ts 对齐
type Vuln struct {
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	Severity          string   `json:"severity"`
	CVE               []string `json:"cve"`
	CWE               []string `json:"cwe"`
	Description       string   `json:"description"`
	Suggestion        string   `json:"suggestion"`
	References        []string `json:"references"`
	ComponentName     string   `json:"componentName"`
	ComponentVersion  string   `json:"componentVersion"`
	ComponentLanguage string   `json:"componentLanguage"`
	Purl              string   `json:"purl"`
	Source            string   `json:"source"`
	// v3.x 新增：发布日期（CLI 原 release_date，例 "2023-10-24"）和利用难度
	//（CLI 原 exploit_level_id：1=易 2=中 3=难 0/缺省=未知）
	ReleaseDate  string `json:"releaseDate"`
	ExploitLevel int    `json:"exploitLevel"`
}

type Component struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Language string `json:"language"`
	Purl     string `json:"purl"`
	Vulns    []Vuln `json:"vulns"`
	// v3.x 新增：许可证（CLI 原 licenses: [{name:"Apache-2.0"}, ...]，
	// 这里只取 name 字符串数组，方便前端直接展示）和依赖方式（CLI 原 direct: bool，
	// true=直接依赖 / false=间接依赖）。
	Licenses []string `json:"licenses"`
	Direct   bool     `json:"direct"`
}

type Report struct {
	TaskID          string            `json:"taskId"`
	GeneratedAt      int64             `json:"generatedAt"`
	TotalComponents int               `json:"totalComponents"`
	TotalVulns      int               `json:"totalVulns"`
	SeverityCount   map[string]int    `json:"severityCount"`
	Components      []Component       `json:"components"`
	// Warning 透传 CLI 自身在 task_info.error 里给的状态消息（v3.x）。
	// 典型场景：未配置漏洞库 → "not config vuln database origin"。
	// 透传而非丢弃，让 UI 能把"为什么没漏洞"展示给用户。
	Warning         string            `json:"warning"`
}

// cliRawVuln opensca-cli 原始 JSON 中的漏洞结构（不同版本字段名差异巨大）。
//
// v2.x 字段：id/title/severity(string)/description/solution/version
// v3.x 字段（实测 XmirrorSecurity/OpenSCA-cli v3.0.11）：
//
//	{
//	  "id": "XMIRROR-xxx",
//	  "name": "漏洞中文标题",
//	  "cve_id": "CVE-2023-46120",
//	  "cnnvd_id": "CNNVD-...",
//	  "cnvd_id": "CNVD-...",
//	  "cwe_id": "CWE-400",
//	  "description": "...",
//	  "suggestion": "...",
//	  "attack_type": "远程",
//	  "release_date": "2023-10-24",
//	  "security_level_id": 3,        // 1=Critical, 2=High, 3=Medium, 4=Low
//	  "exploit_level_id": 0
//	}
//
// 这里把所有 v3.x + v2.x 字段都列上，零值兜底（omitempty 也行但更易读）。
type cliRawVuln struct {
	// v2.x
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Solution    string          `json:"solution"`
	Severity    json.RawMessage `json:"severity"`
	Source      json.RawMessage `json:"source"`
	// v3.x
	Name             string `json:"name"`               // 标题（中文）
	CveID            string `json:"cve_id"`
	CnnvdID          string `json:"cnnvd_id"`
	CnvdID           string `json:"cnvd_id"`
	CweID            string `json:"cwe_id"`
	Suggestion       string `json:"suggestion"`
	AttackType       string `json:"attack_type"`
	ReleaseDate      string `json:"release_date"`
	SecurityLevelID  int    `json:"security_level_id"`  // v3.x：1..4 整数级别
	ExploitLevelID   int    `json:"exploit_level_id"`
}

// cliRawComponent 是 opensca-cli JSON 中的 dependency 元素。
type cliRawComponent struct {
	// v2.x
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Language    string         `json:"language"`
	Purl        string         `json:"purl"`
	Vulnerabilities []cliRawVuln `json:"vulnerabilities"`
	Vulns       []cliRawVuln   `json:"vulns"`
	// v3.x：组件用 vendor+name+version 描述，漏洞在另一个文件 / DB 查询阶段才补上，
	// 所以 v3.x 的 child 通常没有 vulnerabilities 字段。
	Vendor   string   `json:"vendor"`
	Direct   bool     `json:"direct"`
	Paths    []string `json:"paths"`
	Licenses []struct {
		Name string `json:"name"`
	} `json:"licenses"`
}

// cliRawTaskInfo v3.x 顶层 task_info 块（携带 CLI 自身的状态/错误）。
type cliRawTaskInfo struct {
	ToolVersion string `json:"tool_version"`
	AppName     string `json:"app_name"`
	Size        int    `json:"size"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	CostTime    any    `json:"cost_time"` // 数字 or 字符串
	Error       string `json:"error"`
}

// cliRawReport 是 opensca-cli 输出 JSON 的顶层结构。
//
// 顶层候选 key 兼容 v2.x（dependencies / components / results）和 v3.x（children）。
// v3.x 的 children 节点没有 vulnerabilities 字段——v3.x 把漏洞数据放到另一个 json
// 报告里（或通过 -login / 本地 config 走云/本地漏洞库），所以这里解析出的
// 组件数 != 0，漏洞数 = 0 是 v3.x + 无漏洞库时的正常状态。
type cliRawReport struct {
	// v2.x 顶层候选
	Dependencies []cliRawComponent `json:"dependencies"`
	Components   []cliRawComponent `json:"components"`
	Results      []cliRawComponent `json:"results"`
	// v3.x 顶层
	Children []cliRawComponent `json:"children"`
	TaskInfo cliRawTaskInfo     `json:"task_info"`
	ID       string            `json:"id"`
}

// ParseReport 从 JSON 文件解析为前端友好的 Report。
//
// 因为不同 opensca-cli 版本字段名差异较大，这里采用"尽量容忍"的策略：
//   - children / dependencies / components / results 四个候选顶层 key 都尝试
//   - vulns / vulnerabilities 两个候选 key 都尝试（v3.x 不会出现，保持兼容）
//   - 严重度字段可能是字符串或对象，统一转字符串
//   - 顶层 task_info.error（v3.x）原样塞到 Report.Warning，UI 上可见
func ParseReport(taskID, jsonPath string) (*Report, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("read report: %w", err)
	}
	var raw cliRawReport
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse report: %w", err)
	}

	// 选取有内容的源（v3.x 优先匹配 children）
	src := raw.Children
	if len(src) == 0 {
		src = raw.Dependencies
	}
	if len(src) == 0 {
		src = raw.Components
	}
	if len(src) == 0 {
		src = raw.Results
	}

	rpt := &Report{
		TaskID:        taskID,
		GeneratedAt:   time.Now().UnixMilli(),
		SeverityCount: map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0, "info": 0, "unknown": 0},
	}
	// v3.x 的 task_info 错误透传到 UI（典型：未配置漏洞库）
	if raw.TaskInfo.Error != "" {
		rpt.Warning = raw.TaskInfo.Error
	}
	for _, c := range src {
		vulns := c.Vulnerabilities
		if len(vulns) == 0 {
			vulns = c.Vulns
		}
		comp := Component{
			Name:     pickComponentName(c),
			Version:  c.Version,
			Language: c.Language,
			Purl:     c.Purl,
			Vulns:    make([]Vuln, 0, len(vulns)),
			Direct:   c.Direct,
			Licenses: extractLicenseNames(c.Licenses),
		}
		// v3.x 没显式 purl，从 vendor/name/version 拼一个 generic purl 出来，
		// 让 UI 至少能展示一致的组件标识。
		if comp.Purl == "" && comp.Name != "" {
			comp.Purl = buildPurlFromRaw(c)
		}
		for _, v := range vulns {
			comp.Vulns = append(comp.Vulns, normalizeVuln(v, c))
		}
		rpt.TotalVulns += len(comp.Vulns)
		for _, v := range comp.Vulns {
			rpt.SeverityCount[v.Severity]++
		}
		rpt.Components = append(rpt.Components, comp)
	}
	rpt.TotalComponents = len(rpt.Components)
	return rpt, nil
}

// pickComponentName 兼容 v2.x（name）和 v3.x（vendor + name）。
// v3.x 的 Java 组件 vendor 通常是 groupId（com.hdec），name 是 artifactId。
// 显示上用 "vendor:name" 组合更可读。
func pickComponentName(c cliRawComponent) string {
	if c.Name != "" && c.Vendor != "" {
		return c.Vendor + ":" + c.Name
	}
	return c.Name
}

// buildPurlFromRaw 在 v3.x 没有 purl 字段时，从 vendor/name/version 拼一个 generic purl。
// 形式：pkg:generic/<vendor>:<name>@<version>（vendor 缺省时退化为 pkg:generic/<name>@<version>）。
func buildPurlFromRaw(c cliRawComponent) string {
	if c.Name == "" {
		return ""
	}
	id := c.Name
	if c.Vendor != "" {
		id = c.Vendor + ":" + c.Name
	}
	p := "pkg:generic/" + id
	if c.Version != "" {
		p += "@" + c.Version
	}
	return p
}

func normalizeVuln(v cliRawVuln, c cliRawComponent) Vuln {
	// ----- 标题：v2.x=title, v3.x=name -----
	title := v.Title
	if title == "" {
		title = v.Name
	}

	// ----- 严重度：v3.x 是数字 security_level_id，v2.x 是字符串 -----
	sev := ""
	if v.SecurityLevelID > 0 {
		sev = severityFromLevelID(v.SecurityLevelID)
	}
	if sev == "" && len(v.Severity) > 0 {
		// 试字符串
		var s string
		if err := json.Unmarshal(v.Severity, &s); err == nil {
			sev = normalizeSeverity(s)
		} else {
			// 试对象 {value:"high"} 或 {level:"high"}
			var obj map[string]any
			if err := json.Unmarshal(v.Severity, &obj); err == nil {
				for _, k := range []string{"value", "level", "severity"} {
					if val, ok := obj[k]; ok {
						if str, ok := val.(string); ok {
							sev = normalizeSeverity(str)
							break
						}
					}
				}
			}
		}
	}
	if sev == "" {
		sev = "unknown"
	}

	// ----- CVE / CWE 列表 -----
	// v3.x 有显式 cve_id / cnnvd_id / cnvd_id / cwe_id 字段；v2.x 没结构化字段，
	// 从 description 里启发式提取。
	var cve []string
	if v.CveID != "" {
		cve = append(cve, v.CveID)
	}
	if v.CnnvdID != "" {
		cve = append(cve, v.CnnvdID)
	}
	if v.CnvdID != "" {
		cve = append(cve, v.CnvdID)
	}
	if len(cve) == 0 {
		cve, _ = splitCVEs(v.Description)
	}
	var cwe []string
	if v.CweID != "" {
		cwe = append(cwe, v.CweID)
	}

	// ----- source 字段（v2.x 是 RawMessage；v3.x 没有） -----
	source := ""
	if len(v.Source) > 0 {
		var s string
		if err := json.Unmarshal(v.Source, &s); err == nil {
			source = s
		}
	}
	if source == "" && v.AttackType != "" {
		source = v.AttackType
	}

	// ----- 解决方案：v2.x=solution, v3.x=suggestion -----
	suggestion := v.Solution
	if suggestion == "" {
		suggestion = v.Suggestion
	}

	return Vuln{
		ID:                v.ID,
		Title:             title,
		Severity:          sev,
		CVE:               cve,
		CWE:               cwe,
		Description:       v.Description,
		Suggestion:        suggestion,
		ComponentName:     c.Name,
		ComponentVersion:  c.Version,
		ComponentLanguage: c.Language,
		Purl:              c.Purl,
		Source:            source,
		ReleaseDate:       v.ReleaseDate,
		ExploitLevel:      v.ExploitLevelID,
	}
}

// severityFromLevelID 把 v3.x 的整数级别映射到 critical/high/medium/low。
//
// XmirrorSecurity/OpenSCA-cli v3.x SecurityLevel：
//   1 = Critical, 2 = High, 3 = Medium, 4 = Low
// 0 / 越界 → unknown
func severityFromLevelID(id int) string {
	switch id {
	case 1:
		return "critical"
	case 2:
		return "high"
	case 3:
		return "medium"
	case 4:
		return "low"
	}
	return ""
}

// splitCVEs 从描述里按 CVE-/CNNVD-/CNVD- 前缀拆出所有 token。v2.x 没结构化字段时用。
func splitCVEs(desc string) ([]string, []string) {
	descU := strings.ToUpper(desc)
	seen := map[string]struct{}{}
	var out []string
	for _, prefix := range []string{"CVE-", "CNNVD-", "CNVD-"} {
		i := 0
		for i < len(descU) {
			j := strings.Index(descU[i:], prefix)
			if j < 0 {
				break
			}
			i += j
			end := i + len(prefix)
			for end < len(descU) && isCveCweChar(descU[end]) {
				end++
			}
			tok := descU[i:end]
			if _, ok := seen[tok]; !ok {
				seen[tok] = struct{}{}
				out = append(out, tok)
			}
			i = end
		}
	}
	return out, nil
}

// normalizeSeverity 把各种表述映射到 critical/high/medium/low/info/unknown。
func normalizeSeverity(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "critical", "严重", "crit", "c":
		return "critical"
	case "high", "高", "h":
		return "high"
	case "medium", "中", "mid", "m":
		return "medium"
	case "low", "低", "l":
		return "low"
	case "info", "informational", "提示":
		return "info"
	}
	return "unknown"
}

// extractCveCwe 从 description 中启发式提取 CVE-xxxx-xxxx 和 CWE-xxx。
func extractCveCwe(desc string) (cve []string, cwe []string) {
	desc = strings.ToUpper(desc)
	seen := map[string]bool{}
	for _, prefix := range []string{"CVE-", "CWE-"} {
		i := 0
		for i < len(desc) {
			j := strings.Index(desc[i:], prefix)
			if j < 0 {
				break
			}
			i += j
			end := i + len(prefix)
			for end < len(desc) && (isCveCweChar(desc[end])) {
				end++
			}
			tok := desc[i:end]
			if !seen[tok] {
				seen[tok] = true
				if strings.HasPrefix(tok, "CVE-") {
					cve = append(cve, tok)
				} else {
					cwe = append(cwe, tok)
				}
			}
			i = end
		}
	}
	return cve, cwe
}

func isCveCweChar(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '-'
}

// extractLicenseNames 把 CLI 的 [{name:"Apache-2.0"}, {name:"MIT"}] 拍平成 ["Apache-2.0", "MIT"]。
// 缺 licenses 字段 / 空切片都返回空数组（前端组件会显示 "未知"）。
func extractLicenseNames(items []struct {
	Name string `json:"name"`
}) []string {
	if len(items) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		if it.Name != "" {
			out = append(out, it.Name)
		}
	}
	return out
}

// errEmptyReport 用于表示报告为空。
var errEmptyReport = errors.New("report is empty")

// IsEmptyReportErr 判断是否是空报告错误。
func IsEmptyReportErr(err error) bool {
	return errors.Is(err, errEmptyReport)
}