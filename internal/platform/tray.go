package platform

import (
	"context"
	"fmt"
	"os"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// quitFallbackTimeout bounds how long a quit may take before the process exits
// on its own. The tray menu is the only way to quit once the window is hidden,
// so a stalled quit path (a hung Wails message-loop callback, a blocked log
// write, a stuck process snapshot, a wedged native call, ...) must never leave
// the app running with no way to close it.
const quitFallbackTimeout = 2 * time.Second

// quitStateTimeout bounds the process-state query done when the tray quit is
// clicked. It is normally answered in microseconds.
const quitStateTimeout = time.Second

// AppRef defines the interface TrayManager needs from the main App.
// This avoids a circular import between internal/platform and package main.
type AppRef interface {
	GetLocale() string
	GetCtx() context.Context
	// PrepareQuit marks the app as quitting so the window close handler
	// allows the app to exit instead of only hiding the window.
	PrepareQuit()
	// RunningProcessNames returns the names of processes that are still
	// running. Quitting is blocked while this list is not empty.
	RunningProcessNames() []string
	// NotifyQuitBlocked informs the frontend that the quit was blocked
	// because some processes are still running.
	NotifyQuitBlocked(names []string)
	// DescribeProcesses renders the current process state (status, pid,
	// restarts) for the diagnostic line written around a quit.
	DescribeProcesses() string
	// LogSystemError writes a line to the application system log so tray
	// interactions can be diagnosed from a user's log file.
	LogSystemError(component, message string)
}

// TrayManager manages the system tray icon and menu
type TrayManager struct {
	app             AppRef
	trayIcon        []byte
	trayIconWindows []byte
	mShow           *systray.MenuItem
	mQuit           *systray.MenuItem
	// logCh feeds the diagnostic log writer so tray lines keep their order
	// without any tray action ever waiting on a disk write.
	logCh chan string
	// quitMu guards quitStageName, the last quit step started, which the
	// fallback reports when a quit never completes.
	quitMu        sync.Mutex
	quitStageName string
}

// NewTrayManager creates a new TrayManager
func NewTrayManager(app AppRef, trayIcon []byte, trayIconWindows []byte) *TrayManager {
	t := &TrayManager{
		app:             app,
		trayIcon:        trayIcon,
		trayIconWindows: trayIconWindows,
		logCh:           make(chan string, 128),
	}
	go t.drainLogs()
	return t
}

// drainLogs writes queued tray lines one by one, keeping their order and moving
// the disk write off the tray's own goroutines.
func (t *TrayManager) drainLogs() {
	for message := range t.logCh {
		if t.app == nil {
			continue
		}
		t.app.LogSystemError("tray", message)
	}
}

// getLocalizedLabels returns menu labels based on app locale setting
func (t *TrayManager) getLocalizedLabels() (showLabel, quitLabel string) {
	showLabel = "Show Window"
	quitLabel = "Quit"

	// Check app configuration first
	if t.app != nil {
		if t.app.GetLocale() == "zh" {
			showLabel = "显示界面"
			quitLabel = "退出"
		}
	} else if isChineseLocale() {
		// Fallback to system locale
		showLabel = "显示界面"
		quitLabel = "退出"
	}

	return showLabel, quitLabel
}

// UpdateLanguage updates the menu labels based on current locale
func (t *TrayManager) UpdateLanguage() {
	if t.mShow == nil || t.mQuit == nil {
		return
	}

	showLabel, quitLabel := t.getLocalizedLabels()
	t.mShow.SetTitle(showLabel)
	t.mQuit.SetTitle(quitLabel)
}

// isChineseLocale checks if the system is using Chinese locale
func isChineseLocale() bool {
	// Check common environment variables for locale
	for _, envVar := range []string{"LANG", "LC_ALL", "LC_MESSAGES", "LANGUAGE"} {
		if lang := os.Getenv(envVar); lang != "" {
			lang = strings.ToLower(lang)
			if strings.HasPrefix(lang, "zh") {
				return true
			}
		}
	}
	return false
}

// logSystem queues a tray interaction for the application system log. Lines keep
// their order, and the caller never waits: a slow or hung log write must not be
// able to stall the tray menu, most importantly the quit path these lines exist
// to diagnose. Lines are dropped only if the queue is full, which needs a
// back-log of over a hundred pending writes.
func (t *TrayManager) logSystem(format string, args ...interface{}) {
	if t.app == nil {
		return
	}
	message := fmt.Sprintf(format, args...)
	select {
	case t.logCh <- message:
	default:
	}
}

// logSystemSync writes a tray line immediately. Used right before the process
// ends, when a queued line would be lost.
func (t *TrayManager) logSystemSync(format string, args ...interface{}) {
	if t.app == nil {
		return
	}
	t.app.LogSystemError("tray", fmt.Sprintf(format, args...))
}

// onReady is called when systray is ready
func (t *TrayManager) onReady() {
	// Use different tray icons for different platforms
	// macOS: Black template icon (transparent background)
	// Windows: Colored ICO format icon
	if goruntime.GOOS == "windows" {
		systray.SetIcon(t.trayIconWindows)
	} else {
		systray.SetIcon(t.trayIcon)
	}
	systray.SetTooltip("ProcHub - Process Manager")

	// On Windows a left click on the tray icon shows the main window while a
	// right click shows the context menu (the systray default). On other
	// platforms the left click keeps the default behavior (show the menu).
	if goruntime.GOOS == "windows" {
		// The tapped callback runs inside the tray window procedure on the
		// message loop. restoreMainWindowNative makes cross-thread Win32 calls
		// (ShowWindow/SetForegroundWindow) that can block while the main thread
		// is busy; blocking there would freeze the tray loop and leave the icon
		// unresponsive. Hand the restore to a separate goroutine so the message
		// loop returns immediately.
		systray.SetOnTapped(func() {
			t.logSystem("Tray icon tapped")
			go t.showWindow()
		})
	}

	// Get labels based on app configuration
	showLabel, quitLabel := t.getLocalizedLabels()

	// Create menu items with localized labels
	t.mShow = systray.AddMenuItem(showLabel, "Show the main window")
	systray.AddSeparator()
	t.mQuit = systray.AddMenuItem(quitLabel, "Quit the application")
	t.logSystem("Tray menu ready: %q / %q", showLabel, quitLabel)

	// Handle menu clicks in a goroutine.
	//
	// Two rules matter here:
	//  1. The loop must never block: systray delivers a click with a
	//     non-blocking send, so while a handler runs inside this loop the next
	//     click is dropped. A slow window restore (cross-thread
	//     ShowWindow/SetForegroundWindow) would swallow the following "quit".
	//  2. The loop must never exit: once the window is hidden the tray is the
	//     only way to quit, so a blocked quit (processes still running) must
	//     leave the menu working for later attempts.
	go func() {
		for {
			select {
			case <-t.mShow.ClickedCh:
				t.logSystem("Show window clicked")
				go t.showWindow()
			case <-t.mQuit.ClickedCh:
				go t.quitApp()
			}
		}
	}()
}

// onExit is called when systray is about to exit
func (t *TrayManager) onExit() {
	// Cleanup if needed
}

// ShowMainWindow restores and shows the main window, and brings it to the
// foreground. It also restores the Dock icon on macOS. A window hidden with
// SW_HIDE is not brought back by SW_SHOW alone (Wails' WindowShow uses
// SW_SHOW), so on Windows restoreMainWindowNative() issues SW_RESTORE on the
// real HWND first; WindowUnminimise then handles the minimized state.
func ShowMainWindow(ctx context.Context) {
	if ctx == nil {
		return
	}
	restoreMainWindowNative()
	ShowDockIcon()
	runtime.WindowUnminimise(ctx)
	runtime.WindowShow(ctx)
	runtime.WindowSetAlwaysOnTop(ctx, true)
	runtime.WindowSetAlwaysOnTop(ctx, false)
}

// showWindow shows the main application window and Dock icon
func (t *TrayManager) showWindow() {
	if t.app == nil {
		t.logSystem("Show window: tray has no app reference")
		return
	}
	if t.app.GetCtx() == nil {
		t.logSystem("Show window: no Wails context, doing nothing")
		return
	}
	start := time.Now()
	t.logSystem("Show window: clicked (%s)", DescribeMainWindow())
	ShowMainWindow(t.app.GetCtx())
	t.logSystem("Show window: restored in %s", time.Since(start).Round(time.Millisecond))
}

// quitApp quits the application, but only when every managed process has been
// stopped. If any process is still running the quit is blocked: the window is
// shown and the frontend displays a warning telling the user to stop the
// processes first. This prevents silently killing or leaking running jobs.
func (t *TrayManager) quitApp() {
	t.logSystem("Quit requested from tray menu")

	if t.app == nil {
		t.logSystem("Quit: tray has no app reference, stopping the tray only")
		systray.Quit()
		return
	}

	// The fallback is armed before anything else can block, so every step below
	// (state query, native window restore, the Wails quit hand-off) is covered.
	// A quit that is refused because processes are running stops the guard
	// instead, so the fallback only ever fires for a quit the user could expect
	// to succeed.
	guard := time.AfterFunc(quitFallbackTimeout, func() {
		t.exitWithLog("Quit stalled for %s, forcing process exit (last step: %s)", quitFallbackTimeout, t.quitStage())
	})
	t.setQuitStage("state query")
	// The full snapshot goes out on its own goroutine: it is diagnostic only and
	// must not delay (or be able to delay) the quit.
	go t.logProcessSnapshot()

	queryStart := time.Now()
	names, ok := t.runningNames(quitStateTimeout)
	if !ok {
		// The process state could not be read in time. The user asked to quit and
		// the manager is wedged, so leaving the app unclosable is worse than
		// skipping the running-process guard.
		t.exitWithLog("Quit forced: process state unavailable within %s", quitStateTimeout)
	}
	t.logSystem("Quit: state query returned %d running process(es) %v in %s",
		len(names), names, time.Since(queryStart).Round(time.Millisecond))

	// Block quitting while processes are still running.
	if len(names) > 0 {
		guard.Stop()
		t.logSystem("Quit blocked, %d process(es) still running: %v", len(names), names)
		// Bring the window back so the user can see the warning.
		go t.showWindow()
		t.app.NotifyQuitBlocked(names)
		t.logSystem("Quit blocked: quit-blocked event emitted to the frontend")
		return
	}

	// Allow the window close handler to proceed (instead of only hiding).
	t.setQuitStage("prepare quit")
	t.app.PrepareQuit()
	t.logSystem("Quit: app marked as quitting")

	if ctx := t.app.GetCtx(); ctx != nil {
		// On Windows, ask the window thread's queue to end its message loop
		// directly. This is the same exit Wails performs, minus the hand-off
		// through its invoke message, which is the step observed to stop working
		// after the app had been running for a long time. When the window cannot
		// be found, fall back to Wails' own quit.
		t.setQuitStage("post WM_QUIT to the main window thread")
		posted, detail := PostQuitToMainWindow()
		t.logSystem("Quit: WM_QUIT post %s (%s)", postedVerb(posted), detail)
		if posted {
			return
		}
		t.setQuitStage("runtime.Quit")
		t.logSystem("Quit: falling back to runtime.Quit")
		runtime.Quit(ctx)
		return
	}
	t.logSystem("Quit: no Wails context, stopping the tray only")
	guard.Stop()
	systray.Quit()
}

// postedVerb turns the PostQuitToMainWindow result into the word used in the log.
func postedVerb(posted bool) string {
	if posted {
		return "succeeded"
	}
	return "failed"
}

// setQuitStage records how far the current quit got, so a stalled quit reports
// the step it never came back from.
func (t *TrayManager) setQuitStage(stage string) {
	t.quitMu.Lock()
	t.quitStageName = stage
	t.quitMu.Unlock()
}

// quitStage returns the last quit step started.
func (t *TrayManager) quitStage() string {
	t.quitMu.Lock()
	defer t.quitMu.Unlock()
	return t.quitStageName
}

// logProcessSnapshot records every process entry with its status and pid, so a
// report shows why a quit was blocked (or why the app believed nothing ran).
func (t *TrayManager) logProcessSnapshot() {
	detail := t.app.DescribeProcesses()
	if detail == "" {
		return
	}
	t.logSystem("Quit: %s", detail)
}

// runningNames returns the names of the running processes within timeout. ok is
// false when the query did not answer in time, which happens when the process
// manager is wedged on its own lock.
func (t *TrayManager) runningNames(timeout time.Duration) (names []string, ok bool) {
	done := make(chan []string, 1)
	go func() { done <- t.app.RunningProcessNames() }()

	select {
	case names = <-done:
		return names, true
	case <-time.After(timeout):
		return nil, false
	}
}

// exitWithLog records why the process is being forced to end and then ends it.
// A short watchdog ends the process even when that write hangs: the line exists
// to explain the exit, it must never be able to keep the app alive.
func (t *TrayManager) exitWithLog(format string, args ...interface{}) {
	time.AfterFunc(300*time.Millisecond, func() { os.Exit(0) })
	if t.app != nil {
		t.app.LogSystemError("tray", fmt.Sprintf(format, args...))
	}
	os.Exit(0)
}

// Run starts the systray.
//
// On Windows we use systray.Run() so that the hidden tray window and its
// message loop run on the same OS thread. fyne.io/systray's
// RunWithExternalLoop starts the message loop on a separate goroutine on
// Windows (nativeStart spawns one), and GetMessage only pumps the calling
// thread's queue, so the tray window's messages (icon clicks,
// TaskbarCreated re-registration, etc.) are never dispatched and the tray
// icon is not shown reliably.
//
// On macOS we use RunWithExternalLoop instead of Run so that fyne.io/systray
// does NOT call [NSApp run] / nativeLoop again. Wails already owns the macOS
// main event loop; calling it a second time from a goroutine causes the
// "nextEventMatchingMask should only be called from the Main Thread" panic
// that freezes all AppKit dialogs (NSOpenPanel, NSSavePanel, etc.).
//
// The start() function returned by RunWithExternalLoop creates NSStatusBarWindow
// on macOS, which must be called on the main thread. runTrayStart() handles this
// platform-specifically: on Darwin it uses dispatch_async(main_queue), on other
// platforms it calls start() directly.
func (t *TrayManager) Run() {
	t.logSystem("Tray starting (platform=%s, go=%s)", goruntime.GOOS, goruntime.Version())
	if goruntime.GOOS == "windows" {
		// fyne.io/systray creates the hidden tray window and pumps its message
		// loop with GetMessageW on the calling goroutine. GetMessageW only
		// retrieves messages from the calling OS thread's queue, and Windows
		// delivers the tray window's messages (icon clicks, TaskbarCreated) to
		// the thread that created the window. Run() is started in a new
		// goroutine, and without LockOSThread the Go scheduler may migrate it
		// to another OS thread after a while: GetMessageW then blocks on the
		// wrong thread's queue, so the icon silently stops responding (no
		// left/right click, and no re-register after an explorer restart).
		// LockOSThread keeps the loop on the thread that owns the window
		// (systray's own Windows test locks the thread for the same reason).
		goruntime.LockOSThread()
		systray.Run(t.onReady, t.onExit)
		return
	}
	start, _ := systray.RunWithExternalLoop(t.onReady, t.onExit)
	runTrayStart(start)
}

// Quit stops the systray
func (t *TrayManager) Quit() {
	t.logSystemSync("Tray shutdown requested, stopping the tray icon")
	systray.Quit()
}
