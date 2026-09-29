package control

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"prochub/internal/process"
)

// fakeBackend is an in-memory Backend used to exercise the control server and
// client without a real application.
type fakeBackend struct {
	items   []process.Snapshot
	logs    []string
	cfg     ConfigSummary
	started []string
}

func (f *fakeBackend) Status() StatusInfo {
	return StatusInfo{Version: "test", Platform: "mac", ProcessCount: len(f.items), RunningCount: 1}
}

func (f *fakeBackend) ListProcesses() []process.Snapshot { return f.items }

func (f *fakeBackend) StartProcess(id string) error {
	if _, err := f.find(id); err != nil {
		return err
	}
	f.started = append(f.started, id)
	return nil
}

func (f *fakeBackend) StopProcess(id string) error { return nil }

func (f *fakeBackend) Logs(id string, tail int) ([]string, error) {
	if _, err := f.find(id); err != nil {
		return nil, err
	}
	if tail < len(f.logs) {
		return f.logs[len(f.logs)-tail:], nil
	}
	return f.logs, nil
}

func (f *fakeBackend) Config() ConfigSummary { return f.cfg }

func (f *fakeBackend) UpdateConfig(patch ConfigPatch) error {
	if patch.Theme != nil {
		if *patch.Theme != "light" && *patch.Theme != "dark" {
			return errors.New("bad theme")
		}
		f.cfg.Theme = *patch.Theme
	}
	if patch.Locale != nil {
		f.cfg.Locale = *patch.Locale
	}
	if patch.AutoStart != nil {
		f.cfg.AutoStart = *patch.AutoStart
	}
	if patch.AutoStartWSL != nil {
		f.cfg.AutoStartWSL = *patch.AutoStartWSL
	}
	return nil
}

func (f *fakeBackend) find(id string) (process.Snapshot, error) {
	for _, item := range f.items {
		if item.Definition.ID == id {
			return item, nil
		}
	}
	return process.Snapshot{}, process.ErrNotFound
}

func newTestClient(t *testing.T) (*Client, *fakeBackend) {
	t.Helper()
	dir := t.TempDir()
	backend := &fakeBackend{
		items: []process.Snapshot{
			{Definition: process.Definition{ID: "proc-1", Name: "demo"}, Status: process.StatusRunning},
		},
		logs: []string{"line-1", "line-2", "line-3"},
		cfg:  ConfigSummary{Theme: "light", Locale: "zh", LogDir: "logs"},
	}

	token := "test-token"
	srv := NewServerWithPort(backend, token, 0)
	if _, err := srv.Start(); err != nil {
		t.Fatalf("control server start failed: %v", err)
	}
	t.Cleanup(srv.Stop)

	if err := WriteAuth(dir, AuthInfo{Port: srv.Port(), Token: token}); err != nil {
		t.Fatalf("WriteAuth failed: %v", err)
	}
	client, err := NewClient(dir)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	return client, backend
}

func TestClientAuth(t *testing.T) {
	dir := t.TempDir()
	srv := NewServerWithPort(&fakeBackend{}, "right-token", 0)
	if _, err := srv.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer srv.Stop()

	// Wrong token is rejected by the server, so the client surfaces the error.
	if err := WriteAuth(dir, AuthInfo{Port: srv.Port(), Token: "wrong-token"}); err != nil {
		t.Fatalf("WriteAuth failed: %v", err)
	}
	client, err := NewClient(dir)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if _, err := client.Status(); err == nil {
		t.Fatal("expected an error for a wrong token")
	}

	// A missing auth file means the app is not running.
	if _, err := NewClient(t.TempDir()); !os.IsNotExist(err) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
}

func TestClientProcessLifecycle(t *testing.T) {
	client, backend := newTestClient(t)

	items, err := client.ListProcesses()
	if err != nil {
		t.Fatalf("ListProcesses failed: %v", err)
	}
	if len(items) != 1 || items[0].Definition.ID != "proc-1" {
		t.Fatalf("unexpected process list: %+v", items)
	}

	if err := client.StartProcess("proc-1"); err != nil {
		t.Fatalf("StartProcess failed: %v", err)
	}
	if len(backend.started) != 1 || backend.started[0] != "proc-1" {
		t.Fatalf("backend did not receive start: %v", backend.started)
	}

	if err := client.StopProcess("proc-1"); err != nil {
		t.Fatalf("StopProcess failed: %v", err)
	}

	if err := client.StartProcess("missing"); err == nil {
		t.Fatal("expected an error for an unknown process")
	}
}

func TestClientLogs(t *testing.T) {
	client, _ := newTestClient(t)

	lines, err := client.Logs("proc-1", 2)
	if err != nil {
		t.Fatalf("Logs failed: %v", err)
	}
	if len(lines) != 2 || lines[1] != "line-3" {
		t.Fatalf("unexpected tail lines: %v", lines)
	}
}

func TestClientConfigPatch(t *testing.T) {
	client, _ := newTestClient(t)

	cfg, err := client.Config()
	if err != nil {
		t.Fatalf("Config failed: %v", err)
	}
	if cfg.Theme != "light" || cfg.Locale != "zh" || cfg.AutoStart {
		t.Fatalf("unexpected initial config: %+v", cfg)
	}

	theme := "dark"
	locale := "en"
	on := true
	if err := client.UpdateConfig(ConfigPatch{Theme: &theme, Locale: &locale, AutoStart: &on}); err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	updated, err := client.Config()
	if err != nil {
		t.Fatalf("Config after patch failed: %v", err)
	}
	if updated.Theme != "dark" || updated.Locale != "en" || !updated.AutoStart {
		t.Fatalf("patch not applied: %+v", updated)
	}

	bad := "rainbow"
	if err := client.UpdateConfig(ConfigPatch{Theme: &bad}); err == nil {
		t.Fatal("expected an error for an invalid theme")
	}
}

// TestResponseShape guards the JSON envelope the CLI depends on.
func TestResponseShape(t *testing.T) {
	client, _ := newTestClient(t)
	req, err := http.NewRequest(http.MethodGet, client.baseURL+"/control/config", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("build request failed: %v", err)
	}
	req.Header.Set(TokenHeader, client.token)

	resp, err := client.http.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got %v", payload["ok"])
	}
	if _, has := payload["data"]; !has {
		t.Fatal("expected a data field")
	}
	if status := strconv.Itoa(resp.StatusCode); !strings.HasPrefix(status, "2") {
		t.Fatalf("unexpected status: %s", status)
	}
}
