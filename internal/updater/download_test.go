package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOrderBySpeedPutsReachableFirst(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		w.WriteHeader(http.StatusPartialContent)
	}))
	defer slow.Close()
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPartialContent)
	}))
	defer fast.Close()

	ordered := OrderBySpeed(context.Background(), []string{slow.URL, fast.URL}, "test")
	if len(ordered) != 2 || ordered[0] != fast.URL {
		t.Fatalf("ordered = %v", ordered)
	}
}

func TestDownloadAndVerify(t *testing.T) {
	payload := []byte("hello prochub update")
	sum := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "20")
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	asset := Asset{
		Name:   "ProcHub-test.bin",
		Size:   int64(len(payload)),
		SHA256: hex.EncodeToString(sum[:]),
		URLs:   []string{server.URL},
	}
	dir := t.TempDir()
	path, err := Download(context.Background(), asset, dir, "test", nil)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if path != filepath.Join(dir, asset.Name) {
		t.Fatalf("path = %s", path)
	}
	if err := VerifySHA256(path, asset.SHA256); err != nil {
		t.Fatalf("VerifySHA256: %v", err)
	}

	bad := asset
	bad.SHA256 = "00" + asset.SHA256[2:]
	if _, err := Download(context.Background(), bad, t.TempDir(), "test", nil); err == nil {
		t.Fatal("expected a checksum failure")
	}
}

func TestDownloadRefusesMissingChecksum(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()
	asset := Asset{Name: "x.bin", URLs: []string{server.URL}}
	if _, err := Download(context.Background(), asset, t.TempDir(), "test", nil); err == nil {
		t.Fatal("expected an error for an asset without sha256")
	}
}

func TestDownloadReportsProgress(t *testing.T) {
	payload := make([]byte, 256*1024)
	sum := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	var last int64
	asset := Asset{Name: "p.bin", Size: int64(len(payload)), SHA256: hex.EncodeToString(sum[:]), URLs: []string{server.URL}}
	if _, err := Download(context.Background(), asset, t.TempDir(), "test", func(done, total int64) {
		if total <= 0 {
			t.Errorf("total = %d", total)
		}
		last = done
	}); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if last != int64(len(payload)) {
		t.Fatalf("last progress = %d", last)
	}
}

func TestInstallRefusesWhileProcessesRun(t *testing.T) {
	target := filepath.Join(t.TempDir(), "ProcHub.AppImage")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	payload := []byte("new-binary")
	sum := sha256.Sum256(payload)
	downloaded := filepath.Join(t.TempDir(), "ProcHub-new.AppImage")
	if err := os.WriteFile(downloaded, payload, 0o755); err != nil {
		t.Fatal(err)
	}
	manager := New(Options{
		CurrentVersion:   "0.6.0",
		DataDir:          t.TempDir(),
		RunningProcesses: func() []string { return []string{"demo"} },
	})
	manager.asset = &Asset{Kind: "AppImage", Name: "new", SHA256: hex.EncodeToString(sum[:])}
	manager.filePath = downloaded
	manager.state.State = StateReady

	if _, _, err := manager.Install(context.Background()); err == nil {
		t.Fatal("expected the install to be blocked")
	}

	// With no running processes the AppImage must be replaced.
	manager.opts.RunningProcesses = func() []string { return nil }
	manager.state.State = StateReady
	oldCheck := installDependencyCheck
	installDependencyCheck = nil
	defer func() { installDependencyCheck = oldCheck }()
	t.Setenv("APPIMAGE", target)
	result, state, err := manager.Install(context.Background())
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !result.RestartRequired || state.State != StateRestarting {
		t.Fatalf("result=%+v state=%+v", result, state)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("target = %q", got)
	}
	if _, err := os.Stat(target + ".old"); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
}
