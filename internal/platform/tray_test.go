package platform

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// fakeTrayApp records the calls TrayManager makes into the main App.
type fakeTrayApp struct {
	mu           sync.Mutex
	locale       string
	ctx          context.Context
	running      []string
	prepared     bool
	blockedNames []string
	logs         []string
}

func (f *fakeTrayApp) GetLocale() string             { return f.locale }
func (f *fakeTrayApp) GetCtx() context.Context       { return f.ctx }
func (f *fakeTrayApp) RunningProcessNames() []string { return f.running }

func (f *fakeTrayApp) PrepareQuit() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prepared = true
}

func (f *fakeTrayApp) NotifyQuitBlocked(names []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.blockedNames = names
}

func (f *fakeTrayApp) LogSystemError(component, message string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logs = append(f.logs, component+": "+message)
}

func (f *fakeTrayApp) hasLog(substr string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, line := range f.logs {
		if strings.Contains(line, substr) {
			return true
		}
	}
	return false
}

// TestQuitAppBlockedWhileRunning pins the guard: a quit requested from the tray
// while a process is still running must not mark the app as quitting (which
// would let the window close handler exit), and it must warn the frontend.
func TestQuitAppBlockedWhileRunning(t *testing.T) {
	app := &fakeTrayApp{running: []string{"web", "worker"}}
	tray := NewTrayManager(app, nil, nil)

	tray.quitApp()

	if app.prepared {
		t.Error("quitApp called PrepareQuit while processes were still running")
	}
	if len(app.blockedNames) != 2 {
		t.Fatalf("NotifyQuitBlocked names = %v, want the two running processes", app.blockedNames)
	}
	if !app.hasLog("Quit blocked") {
		t.Errorf("tray log = %v, want a blocked-quit line", app.logs)
	}
}

// TestQuitAppLogsRequest pins the diagnostic line written on every tray quit,
// which is what identified the quit path in user reports.
func TestQuitAppLogsRequest(t *testing.T) {
	app := &fakeTrayApp{running: []string{"web"}}
	tray := NewTrayManager(app, nil, nil)

	tray.quitApp()

	if !app.hasLog("Quit requested from tray menu") {
		t.Errorf("tray log = %v, want a quit request line", app.logs)
	}
}

func TestTrayLabels(t *testing.T) {
	// Keep the system-locale fallback out of the way so only the app locale
	// decides the labels.
	for _, env := range []string{"LANG", "LC_ALL", "LC_MESSAGES", "LANGUAGE"} {
		t.Setenv(env, "")
	}
	tests := []struct {
		locale string
		show   string
		quit   string
	}{
		{locale: "zh", show: "显示界面", quit: "退出"},
		{locale: "en", show: "Show Window", quit: "Quit"},
	}
	for _, tc := range tests {
		app := &fakeTrayApp{locale: tc.locale}
		tray := NewTrayManager(app, nil, nil)
		show, quit := tray.getLocalizedLabels()
		if show != tc.show || quit != tc.quit {
			t.Errorf("locale %q labels = %q / %q, want %q / %q", tc.locale, show, quit, tc.show, tc.quit)
		}
	}
}

// TestLogSystemWithoutApp guards the nil-app path used before Startup wired the
// App into the tray manager.
func TestLogSystemWithoutApp(t *testing.T) {
	tray := NewTrayManager(nil, nil, nil)
	tray.logSystem("no app yet")
}
