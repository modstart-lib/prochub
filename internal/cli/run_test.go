package cli

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"prochub/internal/control"
	"prochub/internal/process"
)

type stubBackend struct {
	cfg      control.ConfigSummary
	started  []string
	running  bool
	wslCalls []string
}

func (s *stubBackend) Status() control.StatusInfo {
	return control.StatusInfo{Version: "test", Platform: "mac", ProcessCount: 1, RunningCount: 1}
}

func (s *stubBackend) ListProcesses() []process.Snapshot {
	return []process.Snapshot{
		{Definition: process.Definition{ID: "proc-1", Name: "demo"}, Status: process.StatusRunning},
	}
}

func (s *stubBackend) StartProcess(id string) error {
	if id != "proc-1" {
		return process.ErrNotFound
	}
	s.started = append(s.started, id)
	return nil
}

func (s *stubBackend) StopProcess(id string) error { return nil }

func (s *stubBackend) Logs(id string, tail int) ([]string, error) {
	if id != "proc-1" {
		return nil, process.ErrNotFound
	}
	return []string{"log-a", "log-b"}, nil
}

func (s *stubBackend) WSLStatus() control.WSLStatusInfo {
	return control.WSLStatusInfo{
		Supported: true,
		Available: true,
		Running:   s.running,
		Version:   "2",
		Distro:    "Ubuntu",
	}
}

func (s *stubBackend) StartWSL() error {
	s.wslCalls = append(s.wslCalls, "start")
	s.running = true
	return nil
}

func (s *stubBackend) StopWSL() error {
	s.wslCalls = append(s.wslCalls, "stop")
	s.running = false
	return nil
}

func (s *stubBackend) RestartWSL() error {
	s.wslCalls = append(s.wslCalls, "restart")
	s.running = true
	return nil
}

func (s *stubBackend) Config() control.ConfigSummary { return s.cfg }

func (s *stubBackend) UpdateConfig(patch control.ConfigPatch) error {
	if patch.Theme != nil {
		if *patch.Theme != "light" && *patch.Theme != "dark" {
			return errors.New("主题仅支持 light 或 dark")
		}
		s.cfg.Theme = *patch.Theme
	}
	if patch.Locale != nil {
		s.cfg.Locale = *patch.Locale
	}
	if patch.AutoStart != nil {
		s.cfg.AutoStart = *patch.AutoStart
	}
	if patch.AutoStartWSL != nil {
		s.cfg.AutoStartWSL = *patch.AutoStartWSL
	}
	return nil
}

// setupApp boots a control server backed by the stub and points the CLI at it.
func setupApp(t *testing.T, backend *stubBackend) Options {
	t.Helper()
	dir := t.TempDir()
	token := "cli-token"
	srv := control.NewServerWithPort(backend, token, 0)
	if _, err := srv.Start(); err != nil {
		t.Fatalf("control server start failed: %v", err)
	}
	t.Cleanup(srv.Stop)
	if err := control.WriteAuth(dir, control.AuthInfo{Port: srv.Port(), Token: token}); err != nil {
		t.Fatalf("WriteAuth failed: %v", err)
	}
	return Options{DataDir: dir, Version: "0.6.0", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
}

func newBackend() *stubBackend {
	return &stubBackend{cfg: control.ConfigSummary{Theme: "light", Locale: "zh", LogDir: "logs"}}
}

func TestIsCommand(t *testing.T) {
	if !IsCommand([]string{"process", "list"}) {
		t.Fatal("process should be a CLI command")
	}
	if IsCommand(nil) || IsCommand([]string{"--some-flag"}) {
		t.Fatal("empty or unknown args should start the GUI")
	}
}

func TestRunProcessStart(t *testing.T) {
	backend := newBackend()
	opts := setupApp(t, backend)

	if code := Run([]string{"process", "start", "proc-1"}, opts); code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if len(backend.started) != 1 || backend.started[0] != "proc-1" {
		t.Fatalf("start was not forwarded: %v", backend.started)
	}
	if code := Run([]string{"process", "start", "missing"}, opts); code != 1 {
		t.Fatalf("expected exit 1 for an unknown process, got %d", code)
	}
}

func TestRunConfigSetTheme(t *testing.T) {
	backend := newBackend()
	opts := setupApp(t, backend)

	if code := Run([]string{"config", "set", "theme", "dark"}, opts); code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if backend.cfg.Theme != "dark" {
		t.Fatalf("theme was not applied: %s", backend.cfg.Theme)
	}
	if code := Run([]string{"config", "set", "theme", "rainbow"}, opts); code != 1 {
		t.Fatalf("expected exit 1 for an invalid theme, got %d", code)
	}
}

func TestRunConfigSetLanguageAndSwitches(t *testing.T) {
	backend := newBackend()
	opts := setupApp(t, backend)

	if code := Run([]string{"config", "set", "language", "en"}, opts); code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if code := Run([]string{"config", "set", "autostart", "on"}, opts); code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if code := Run([]string{"config", "set", "autostart-wsl", "off"}, opts); code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if backend.cfg.Locale != "en" || !backend.cfg.AutoStart || backend.cfg.AutoStartWSL {
		t.Fatalf("config not applied: %+v", backend.cfg)
	}
}

func TestRunStatusAndLogs(t *testing.T) {
	backend := newBackend()
	opts := setupApp(t, backend)

	if code := Run([]string{"status"}, opts); code != 0 {
		t.Fatalf("expected exit 0 for status, got %d", code)
	}
	out := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "ProcHub") || !strings.Contains(out, "mac") {
		t.Fatalf("unexpected status output: %q", out)
	}

	opts.Stdout.(*bytes.Buffer).Reset()
	if code := Run([]string{"process", "logs", "proc-1", "--tail", "5"}, opts); code != 0 {
		t.Fatalf("expected exit 0 for logs, got %d", code)
	}
	if got := opts.Stdout.(*bytes.Buffer).String(); !strings.Contains(got, "log-b") {
		t.Fatalf("unexpected logs output: %q", got)
	}
}

func TestRunWithoutRunningApp(t *testing.T) {
	opts := Options{DataDir: t.TempDir(), Version: "0.6.0", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	if code := Run([]string{"status"}, opts); code != 1 {
		t.Fatalf("expected exit 1 when the app is not running, got %d", code)
	}
	if got := opts.Stderr.(*bytes.Buffer).String(); !strings.Contains(got, "未运行") {
		t.Fatalf("expected a not-running hint, got %q", got)
	}
}

func TestRunWSLStatusAndActions(t *testing.T) {
	backend := newBackend()
	opts := setupApp(t, backend)

	// start
	if code := Run([]string{"wsl", "start"}, opts); code != 0 {
		t.Fatalf("expected exit 0 for wsl start, got %d; stderr: %s", code, opts.Stderr.(*bytes.Buffer).String())
	}
	if len(backend.wslCalls) != 1 || backend.wslCalls[0] != "start" {
		t.Fatalf("start was not forwarded: %v", backend.wslCalls)
	}

	// status reflects the running state
	opts.Stdout.(*bytes.Buffer).Reset()
	if code := Run([]string{"wsl", "status"}, opts); code != 0 {
		t.Fatalf("expected exit 0 for wsl status, got %d", code)
	}
	out := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "运行中") || !strings.Contains(out, "Ubuntu") {
		t.Fatalf("unexpected wsl status output: %q", out)
	}

	// stop
	if code := Run([]string{"wsl", "stop"}, opts); code != 0 {
		t.Fatalf("expected exit 0 for wsl stop, got %d", code)
	}
	// restart
	if code := Run([]string{"wsl", "restart"}, opts); code != 0 {
		t.Fatalf("expected exit 0 for wsl restart, got %d", code)
	}
	if len(backend.wslCalls) != 3 {
		t.Fatalf("expected 3 wsl calls, got %v", backend.wslCalls)
	}
}

func TestRunWSLUnknownSubcommand(t *testing.T) {
	opts := setupApp(t, newBackend())
	if code := Run([]string{"wsl", "bogus"}, opts); code != 1 {
		t.Fatalf("expected exit 1 for an unknown wsl subcommand, got %d", code)
	}
}

func TestWSLSubcommandsWithoutRunningApp(t *testing.T) {
	opts := Options{DataDir: t.TempDir(), Version: "0.6.0", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	if code := Run([]string{"wsl", "start"}, opts); code != 1 {
		t.Fatalf("expected exit 1 when the app is not running, got %d", code)
	}
}

func TestRunVersionAndHelp(t *testing.T) {
	opts := Options{DataDir: t.TempDir(), Version: "9.9.9", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	if code := Run([]string{"version"}, opts); code != 0 {
		t.Fatalf("expected exit 0 for version, got %d", code)
	}
	if got := opts.Stdout.(*bytes.Buffer).String(); !strings.Contains(got, "9.9.9") {
		t.Fatalf("version output missing: %q", got)
	}

	opts.Stdout.(*bytes.Buffer).Reset()
	if code := Run([]string{"help"}, opts); code != 0 {
		t.Fatalf("expected exit 0 for help, got %d", code)
	}
	if !strings.Contains(opts.Stdout.(*bytes.Buffer).String(), "prochub process") {
		t.Fatal("help output missing usage")
	}

	// The token file is created with owner-only permissions.
	dir := t.TempDir()
	if err := control.WriteAuth(dir, control.AuthInfo{Port: 1, Token: "x"}); err != nil {
		t.Fatalf("WriteAuth failed: %v", err)
	}
	info, err := os.Stat(control.AuthPath(dir))
	if err != nil {
		t.Fatalf("stat auth file failed: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected auth file mode 0600, got %o", info.Mode().Perm())
	}
}
