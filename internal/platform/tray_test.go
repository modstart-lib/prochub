package platform

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
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

func (f *fakeTrayApp) GetLocale() string       { return f.locale }
func (f *fakeTrayApp) GetCtx() context.Context { return f.ctx }

func (f *fakeTrayApp) DescribeProcesses() string {
	return "processes: web(id-1) status=stopped pid=0 restarts=0"
}

func (f *fakeTrayApp) RunningProcessNames() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running
}

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

// waitForLog polls until a matching line appears: tray logging is asynchronous
// so that a hung log write can never stall the tray menu.
func (f *fakeTrayApp) waitForLog(substr string) bool {
	deadline := time.Now().Add(2 * time.Second)
	for {
		if f.hasLog(substr) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// logIndex returns the position of the first matching line, -1 when absent.
func (f *fakeTrayApp) logIndex(substr string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, line := range f.logs {
		if strings.Contains(line, substr) {
			return i
		}
	}
	return -1
}

// blockedApp keeps the process-state query blocked, like a wedged manager.
type blockedApp struct {
	fakeTrayApp
	release chan struct{}
}

func (b *blockedApp) RunningProcessNames() []string {
	<-b.release
	return nil
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
	if !app.waitForLog("Quit blocked") {
		t.Error("want a blocked-quit line in the tray log")
	}
}

// TestQuitAppLogsRequest pins the diagnostic line written on every tray quit,
// which is what identified the quit path in user reports.
func TestQuitAppLogsRequest(t *testing.T) {
	app := &fakeTrayApp{running: []string{"web"}}
	tray := NewTrayManager(app, nil, nil)

	tray.quitApp()

	if !app.waitForLog("Quit requested from tray menu") {
		t.Error("want a quit-request line in the tray log")
	}
}

// TestRunningNamesTimesOut pins that a wedged process manager cannot hold the
// quit path: the query reports failure instead of blocking forever.
func TestRunningNamesTimesOut(t *testing.T) {
	app := &blockedApp{release: make(chan struct{})}
	defer close(app.release)
	tray := NewTrayManager(app, nil, nil)

	start := time.Now()
	names, ok := tray.runningNames(20 * time.Millisecond)

	if ok {
		t.Fatalf("runningNames returned ok with %v, want a timeout", names)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("runningNames took %s, want it to give up after the timeout", elapsed)
	}
}

func TestRunningNames(t *testing.T) {
	app := &fakeTrayApp{running: []string{"web"}}
	tray := NewTrayManager(app, nil, nil)

	names, ok := tray.runningNames(time.Second)

	if !ok || len(names) != 1 || names[0] != "web" {
		t.Fatalf("runningNames() = %v, %v; want [web], true", names, ok)
	}
}

// TestQuitLogOrder pins that the quit diagnostics land in the order the steps
// ran, so a user's log file reads as a timeline instead of a shuffled set.
func TestQuitLogOrder(t *testing.T) {
	app := &fakeTrayApp{running: []string{"web"}}
	tray := NewTrayManager(app, nil, nil)

	tray.quitApp()

	if !app.waitForLog("Quit blocked") {
		t.Fatal("want a blocked-quit line in the tray log")
	}
	request := app.logIndex("Quit requested")
	query := app.logIndex("state query returned")
	blocked := app.logIndex("Quit blocked")
	if request < 0 || query < 0 || blocked < 0 {
		t.Fatalf("missing quit lines: request=%d query=%d blocked=%d", request, query, blocked)
	}
	if request > query || query > blocked {
		t.Errorf("quit log order = request %d, query %d, blocked %d; want ascending", request, query, blocked)
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
