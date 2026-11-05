package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	// DefaultRepo is used by the GitHub fallback when the backend is unreachable.
	DefaultRepo = "modstart-lib/prochub"
)

// Client talks to the update backend and falls back to GitHub Releases.
type Client struct {
	Endpoint  string
	Repo      string
	Platform  Platform
	UserAgent string
	HTTP      *http.Client

	// DisableGitHubFallback keeps Check offline when the backend itself fails.
	// It exists for tests and for callers that want a single source of truth.
	DisableGitHubFallback bool
}

// NewClient creates a client. Endpoint is the full app_manager/updater URL.
func NewClient(endpoint, repo string, platform Platform, userAgent string) *Client {
	if repo == "" {
		repo = DefaultRepo
	}
	if platform.Name == "" || platform.Arch == "" {
		platform = CurrentPlatform()
	}
	return &Client{
		Endpoint:  strings.TrimRight(endpoint, "/"),
		Repo:      repo,
		Platform:  platform,
		UserAgent: userAgent,
		HTTP:      &http.Client{Timeout: 30 * time.Second},
	}
}

// CurrentPlatform maps the Go runtime to the backend's platform vocabulary.
func CurrentPlatform() Platform {
	name := runtime.GOOS
	switch name {
	case "darwin":
		name = "mac"
	case "windows":
		name = "win"
	}
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x64"
	case "386":
		arch = "x86"
	case "aarch64":
		arch = "arm64"
	}
	return Platform{Name: name, Arch: arch, Version: osVersion()}
}

func osVersion() string {
	switch runtime.GOOS {
	case "darwin":
		if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
			return strings.TrimSpace(string(out))
		}
	case "linux":
		if out, err := exec.Command("uname", "-r").Output(); err == nil {
			return strings.TrimSpace(string(out))
		}
	case "windows":
		if out, err := exec.Command("cmd", "/c", "ver").Output(); err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return ""
}

type checkRequest struct {
	UUID     string   `json:"uuid"`
	Version  string   `json:"version"`
	Channel  string   `json:"channel,omitempty"`
	Platform Platform `json:"platform"`
}

type checkEnvelope struct {
	Code int     `json:"code"`
	Msg  string  `json:"msg"`
	Data Release `json:"data"`
}

// Check asks the backend for the newest release. If the backend fails it falls
// back to the public GitHub Releases API.
func (c *Client) Check(ctx context.Context, currentVersion, channel, uuid string) (*Release, error) {
	release, backendErr := c.checkBackend(ctx, currentVersion, channel, uuid)
	if backendErr == nil && release != nil {
		release.Source = "backend"
		return release, nil
	}
	if c.DisableGitHubFallback {
		return nil, backendErr
	}
	release, githubErr := c.checkGitHub(ctx, channel)
	if githubErr == nil && release != nil {
		release.Source = "github"
		return release, nil
	}
	if backendErr != nil {
		return nil, backendErr
	}
	return nil, githubErr
}

func (c *Client) checkBackend(ctx context.Context, currentVersion, channel, uuid string) (*Release, error) {
	if c.Endpoint == "" {
		return nil, fmt.Errorf("update endpoint is empty")
	}
	body, err := json.Marshal(checkRequest{
		UUID:     uuid,
		Version:  currentVersion,
		Channel:  channel,
		Platform: c.Platform,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update backend request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update backend HTTP %d", resp.StatusCode)
	}
	var envelope checkEnvelope
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("update backend response: %w", err)
	}
	if envelope.Code != 0 {
		return nil, fmt.Errorf("update backend error %d: %s", envelope.Code, envelope.Msg)
	}
	if envelope.Data.Version == "" {
		return nil, fmt.Errorf("update backend returned no version")
	}
	if !IsSemanticVersion(envelope.Data.Version) {
		return nil, fmt.Errorf("update backend returned non-semantic version %q", envelope.Data.Version)
	}
	return &envelope.Data, nil
}

type githubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt string        `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

func (c *Client) checkGitHub(ctx context.Context, channel string) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=30", c.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github releases request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases HTTP %d", resp.StatusCode)
	}
	var releases []githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&releases); err != nil {
		return nil, fmt.Errorf("github releases response: %w", err)
	}

	var best *githubRelease
	for i := range releases {
		item := &releases[i]
		if item.Draft {
			continue
		}
		if channel != "beta" && item.Prerelease {
			continue
		}
		if !IsSemanticVersion(item.TagName) {
			continue
		}
		if best == nil || CompareVersions(item.TagName, best.TagName) > 0 {
			best = item
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no usable github release")
	}

	release := &Release{
		Name:    best.Name,
		Version: best.TagName,
		Summary: summaryOf(best.Body),
		Notes:   best.Body,
		Channel: channel,
	}
	if best.PublishedAt != "" {
		if t, err := time.Parse(time.RFC3339, best.PublishedAt); err == nil {
			release.Time = t.Format("2006-01-02")
		}
	}

	checksums := map[string]string{}
	for _, raw := range best.Assets {
		if strings.EqualFold(raw.Name, "checksums.txt") {
			checksums = c.fetchChecksums(ctx, raw.BrowserDownloadURL)
		}
	}
	for _, raw := range best.Assets {
		parsed := parseAssetName(raw.Name)
		if parsed == nil {
			continue
		}
		if !c.matchesPlatform(parsed.os, parsed.arch) {
			continue
		}
		sum := checksums[raw.Name]
		if sum == "" && strings.HasPrefix(raw.Digest, "sha256:") {
			sum = strings.TrimPrefix(raw.Digest, "sha256:")
		}
		release.Assets = append(release.Assets, Asset{
			OS:     parsed.os,
			Arch:   parsed.arch,
			Kind:   parsed.kind,
			Name:   raw.Name,
			Size:   raw.Size,
			SHA256: sum,
			URLs:   []string{raw.BrowserDownloadURL},
		})
	}
	if len(release.Assets) == 0 {
		return nil, fmt.Errorf("no github asset for %s/%s", c.Platform.Name, c.Platform.Arch)
	}
	return release, nil
}

func (c *Client) fetchChecksums(ctx context.Context, url string) map[string]string {
	out := map[string]string{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return out
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return out
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 || len(fields[0]) != 64 {
			continue
		}
		out[strings.TrimPrefix(fields[1], "*")] = strings.ToLower(fields[0])
	}
	return out
}

type assetNameParts struct {
	os   string
	arch string
	kind string
}

func parseAssetName(name string) *assetNameParts {
	// ProcHub-<version>-<os>-<arch>.<ext>
	ext := ""
	switch {
	case strings.HasSuffix(name, ".AppImage"):
		ext = "AppImage"
	case strings.HasSuffix(name, ".deb"):
		ext = "deb"
	case strings.HasSuffix(name, ".dmg"):
		ext = "dmg"
	case strings.HasSuffix(name, ".zip"):
		ext = "zip"
	case strings.HasSuffix(name, ".exe"):
		ext = "exe"
	default:
		return nil
	}
	base := strings.TrimSuffix(name, "."+ext)
	parts := strings.Split(base, "-")
	if len(parts) < 4 || parts[0] != "ProcHub" {
		return nil
	}
	arch := parts[len(parts)-1]
	osName := parts[len(parts)-2]
	if arch != "amd64" && arch != "arm64" {
		return nil
	}
	if osName != "darwin" && osName != "windows" && osName != "linux" {
		return nil
	}
	return &assetNameParts{os: osName, arch: arch, kind: ext}
}

func (c *Client) matchesPlatform(osName, arch string) bool {
	wantOS := map[string]string{"mac": "darwin", "darwin": "darwin", "win": "windows", "windows": "windows", "linux": "linux"}[c.Platform.Name]
	wantArch := map[string]string{"x64": "amd64", "amd64": "amd64", "arm64": "arm64"}[c.Platform.Arch]
	return (wantOS == "" || wantOS == osName) && (wantArch == "" || wantArch == arch)
}

func summaryOf(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			if len(line) > 200 {
				line = line[:200]
			}
			return line
		}
	}
	return ""
}

// PickAsset chooses the best asset for the current platform. macOS prefers zip
// because a running .app cannot be replaced from a dmg without mounting it.
func PickAsset(release *Release, platform Platform) *Asset {
	if release == nil {
		return nil
	}
	// Only formats the installers can replace in place. dmg stays a manual
	// download and deb belongs to the system package manager.
	priority := map[string]int{"zip": 1, "exe": 1, "AppImage": 1}
	best := -1
	for i := range release.Assets {
		asset := &release.Assets[i]
		rank, ok := priority[asset.Kind]
		if !ok {
			continue
		}
		if best == -1 || rank < priority[release.Assets[best].Kind] {
			best = i
		}
	}
	if best == -1 {
		return nil
	}
	return &release.Assets[best]
}
