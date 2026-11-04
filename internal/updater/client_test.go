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
