package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientCheckBackend(t *testing.T) {
	var got checkRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"msg":  "ok",
			"data": Release{
				Version: "v0.7.0",
				Assets: []Asset{{
					OS: "linux", Arch: "amd64", Kind: "AppImage",
					Name: "ProcHub-0.7.0-linux-amd64.AppImage",
					Size: 3, SHA256: "abc", URLs: []string{"http://example/a"},
				}},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "modstart-lib/prochub", Platform{Name: "linux", Arch: "x64"}, "test")
	release, err := client.Check(context.Background(), "0.6.0", "stable", "uuid-1")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if release.Version != "v0.7.0" || release.Source != "backend" {
		t.Fatalf("release = %+v", release)
	}
	if got.Platform.Name != "linux" || got.Platform.Arch != "x64" || got.Version != "0.6.0" {
		t.Fatalf("request = %+v", got)
	}
	if len(release.Assets) != 1 || release.Assets[0].Kind != "AppImage" {
		t.Fatalf("assets = %+v", release.Assets)
	}
}

func TestClientRejectsNonSemverBackend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"data": Release{Version: "vsnapshot"},
		})
	}))
	defer server.Close()
	client := NewClient(server.URL, "modstart-lib/prochub", Platform{Name: "linux", Arch: "x64"}, "test")
	client.DisableGitHubFallback = true
	if _, err := client.Check(context.Background(), "0.6.0", "stable", ""); err == nil {
		t.Fatal("expected an error for a non-semantic version")
	}
}

func TestParseAssetName(t *testing.T) {
	parts := parseAssetName("ProcHub-0.8.0-beta-darwin-arm64.zip")
	if parts == nil || parts.os != "darwin" || parts.arch != "arm64" || parts.kind != "zip" {
		t.Fatalf("parts = %+v", parts)
	}
	// A numeric pre-release suffix must not confuse the os/arch split.
	parts = parseAssetName("ProcHub-0.8.0-beta.1-linux-amd64.AppImage")
	if parts == nil || parts.os != "linux" || parts.arch != "amd64" || parts.kind != "AppImage" {
		t.Fatalf("parts = %+v", parts)
	}
	if parseAssetName("ProcHub-linux-amd64.tar.gz") != nil {
		t.Fatal("unsupported archive should be ignored")
	}
}

func TestPickAssetPrefersZip(t *testing.T) {
	release := &Release{Assets: []Asset{
		{Kind: "dmg", Name: "a.dmg"},
		{Kind: "zip", Name: "a.zip"},
	}}
	asset := PickAsset(release, Platform{Name: "mac", Arch: "arm64"})
	if asset == nil || asset.Kind != "zip" {
		t.Fatalf("asset = %+v", asset)
	}
}

func TestPickAssetFiltersPlatform(t *testing.T) {
	release := &Release{Assets: []Asset{
		{OS: "mac", Arch: "arm64", Kind: "zip", Name: "mac.zip"},
		{OS: "linux", Arch: "x64", Kind: "AppImage", Name: "linux.AppImage"},
		{OS: "windows", Arch: "x64", Kind: "exe", Name: "win.exe"},
	}}
	if asset := PickAsset(release, Platform{Name: "linux", Arch: "x64"}); asset == nil || asset.Name != "linux.AppImage" {
		t.Fatalf("linux asset = %+v", asset)
	}
	if asset := PickAsset(release, Platform{Name: "win", Arch: "amd64"}); asset == nil || asset.Name != "win.exe" {
		t.Fatalf("windows asset = %+v", asset)
	}
	if asset := PickAsset(release, Platform{Name: "mac", Arch: "arm64"}); asset == nil || asset.Name != "mac.zip" {
		t.Fatalf("mac asset = %+v", asset)
	}
	if asset := PickAsset(release, Platform{Name: "linux", Arch: "arm64"}); asset != nil {
		t.Fatalf("unexpected asset = %+v", asset)
	}
}

// TestClientGitHubFallbackWithChecksums covers the path used when the update
// backend is unreachable: the newest published GitHub release is selected and
// its sha256 is read from checksums.txt.
func TestClientGitHubFallbackWithChecksums(t *testing.T) {
	const sha = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/modstart-lib/prochub/releases":
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"tag_name":     "v0.9.0",
					"name":         "ProcHub 0.9.0",
					"body":         "# Notes\nsearch fixes",
					"html_url":     originOf(r) + "/release/v0.9.0",
					"draft":        false,
					"prerelease":   false,
					"published_at": "2026-11-01T10:00:00Z",
					"assets": []map[string]interface{}{
						{"name": "checksums.txt", "browser_download_url": originOf(r) + "/checksums.txt"},
						{"name": "ProcHub-0.9.0-linux-amd64.AppImage", "size": 3, "browser_download_url": originOf(r) + "/asset"},
					},
				},
				{
					"tag_name":   "v0.10.0-beta",
					"name":       "beta",
					"draft":      false,
					"prerelease": true,
				},
			})
		case "/checksums.txt":
			_, _ = w.Write([]byte(sha + "  ProcHub-0.9.0-linux-amd64.AppImage\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL+"/missing", "modstart-lib/prochub", Platform{Name: "linux", Arch: "x64"}, "test")
	client.GitHubAPI = server.URL + "/repos"
	release, err := client.Check(context.Background(), "0.8.0", "stable", "")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if release.Source != "github" || release.Version != "v0.9.0" {
		t.Fatalf("release = %+v", release)
	}
	if release.Summary != "search fixes" || release.URL == "" {
		t.Fatalf("summary/url = %q / %q", release.Summary, release.URL)
	}
	if len(release.Assets) != 1 {
		t.Fatalf("assets = %+v", release.Assets)
	}
	asset := release.Assets[0]
	if asset.SHA256 != sha || asset.Kind != "AppImage" || asset.Name != "ProcHub-0.9.0-linux-amd64.AppImage" {
		t.Fatalf("asset = %+v", asset)
	}
	if len(asset.URLs) != 1 || asset.URLs[0] != server.URL+"/asset" {
		t.Fatalf("urls = %v", asset.URLs)
	}
}

func originOf(r *http.Request) string {
	return "http://" + r.Host
}
