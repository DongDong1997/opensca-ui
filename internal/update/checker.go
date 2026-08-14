package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// Info 是给前端展示的更新信息。
type Info struct {
	HasUpdate      bool   `json:"hasUpdate"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseName    string `json:"releaseName"`
	ReleaseURL     string `json:"releaseURL"`
	Changelog      string `json:"changelog"`
	DownloadURL    string `json:"downloadURL"`
	AssetName      string `json:"assetName"`
	PublishedAt    string `json:"publishedAt"`
	Message        string `json:"message"`
}

// RepoOwner/Repo 是被检查的项目（opensca-cli 在 XmirrorSecurity 组织下）。
const (
	RepoOwner = "XmirrorSecurity"
	RepoName  = "OpenSCA-cli"
)

// APIURL 是 GitHub releases latest 接口。
const APIURL = "https://api.github.com/repos/" + RepoOwner + "/" + RepoName + "/releases/latest"

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	HTMLURL     string        `json:"html_url"`
	PublishedAt string        `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// CheckUpdate 查询 GitHub latest release，按当前 OS/arch 挑下载资产。
//
// 优先直连 GitHub，失败则走 ghproxy.com 镜像（国内网络更稳）。
func CheckUpdate(currentVersion string) (Info, error) {
	mirrors := []string{
		APIURL,
		"https://ghproxy.com/" + APIURL,
		"https://mirror.ghproxy.com/" + APIURL,
	}
	var lastErr error
	for _, url := range mirrors {
		info, err := tryGitHubAPI(url, currentVersion)
		if err == nil {
			return info, nil
		}
		lastErr = err
	}
	return Info{CurrentVersion: currentVersion, Message: "无法连接 GitHub：" + lastErr.Error()}, lastErr
}

func tryGitHubAPI(url, currentVersion string) (Info, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Info{}, err
	}
	req.Header.Set("User-Agent", "opensca-ui/0.1.0")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return Info{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return Info{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return Info{}, err
	}
	var rel githubRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return Info{}, err
	}
	if rel.TagName == "" {
		return Info{}, errors.New("GitHub 响应缺少 tag_name")
	}

	info := Info{
		CurrentVersion: currentVersion,
		LatestVersion:  strings.TrimPrefix(rel.TagName, "v"),
		ReleaseName:    rel.Name,
		ReleaseURL:     rel.HTMLURL,
		Changelog:      rel.Body,
		PublishedAt:    rel.PublishedAt,
	}

	keyword := platformKeyword()
	var fallback *githubAsset
	for i, a := range rel.Assets {
		lower := strings.ToLower(a.Name)
		if strings.Contains(lower, keyword) && strings.HasSuffix(lower, ".zip") {
			info.AssetName = a.Name
			info.DownloadURL = a.BrowserDownloadURL
			break
		}
		if fallback == nil && strings.HasSuffix(lower, ".zip") {
			fallback = &rel.Assets[i]
		}
	}
	if info.DownloadURL == "" && fallback != nil {
		info.AssetName = fallback.Name
		info.DownloadURL = fallback.BrowserDownloadURL
	}
	if info.ReleaseURL == "" {
		info.ReleaseURL = "https://github.com/" + RepoOwner + "/" + RepoName + "/releases/latest"
	}

	info.HasUpdate = compareVersion(currentVersion, info.LatestVersion) < 0
	if !info.HasUpdate {
		info.Message = "已是最新版本"
	} else if info.DownloadURL == "" {
		info.Message = "发现新版本但未找到匹配的下载资产，请前往 release 页面手动下载"
	}
	return info, nil
}

func platformKeyword() string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	switch os {
	case "windows":
		if arch == "amd64" {
			return "windows-amd64"
		}
		if arch == "arm64" {
			return "windows-arm64"
		}
		return "windows"
	case "darwin":
		if arch == "arm64" {
			return "darwin-arm64"
		}
		return "darwin-amd64"
	default:
		if arch == "arm64" {
			return "linux-arm64"
		}
		return "linux-amd64"
	}
}

// compareVersion 简单语义化版本比较（不处理 pre-release 后缀）。
// 返回 -1 / 0 / 1。
func compareVersion(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	stripSuffix := func(s string) string {
		if i := strings.IndexAny(s, "-+"); i >= 0 {
			return s[:i]
		}
		return s
	}
	a = stripSuffix(a)
	b = stripSuffix(b)
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	n := len(ap)
	if len(bp) > n {
		n = len(bp)
	}
	for i := 0; i < n; i++ {
		var ai, bi int
		if i < len(ap) {
			fmt.Sscanf(ap[i], "%d", &ai)
		}
		if i < len(bp) {
			fmt.Sscanf(bp[i], "%d", &bi)
		}
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}
	return 0
}
