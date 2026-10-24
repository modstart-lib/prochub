package platform

import (
	"context"
	"fmt"
	"os"
	goruntime "runtime"
	"strings"
	"time"

	"fyne.io/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// quitFallbackTimeout bounds how long the graceful quit may take before the
// process exits on its own. The tray menu is the only way to quit once the
// window is hidden, so a stalled Wails quit path must not leave the app
// running without any way to close it.
const quitFallbackTimeout = 3 * time.Second

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
}

// NewTrayManager creates a new TrayManager
func NewTrayManager(app AppRef, trayIcon []byte, trayIconWindows []byte) *TrayManager {
	return &TrayManager{
		app:             app,
		trayIcon:        trayIcon,
		trayIconWindows: trayIconWindows,
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

// logSystem writes a tray interaction into the application system log.
func (t *TrayManager) logSystem(format string, args ...interface{}) {
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
		systray.SetOnTapped(func() { go t.showWindow() })
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
		return
	}
	ShowMainWindow(t.app.GetCtx())
}

// quitApp quits the application, but only when every managed process has been
// stopped. If any process is still running the quit is blocked: the window is
// shown and the frontend displays a warning telling the user to stop the
// processes first. This prevents silently killing or leaking running jobs.
func (t *TrayManager) quitApp() {
	t.logSystem("Quit requested from tray menu")

	if t.app == nil {
		systray.Quit()
		return
	}

	// Block quitting while processes are still running.
	if names := t.app.RunningProcessNames(); len(names) > 0 {
		t.logSystem("Quit blocked, %d process(es) still running: %v", len(names), names)
		// Bring the window back so the user can see the warning.
		t.showWindow()
		t.app.NotifyQuitBlocked(names)
		return
	}

	// Allow the window close handler to proceed (instead of only hiding).
	t.app.PrepareQuit()

	// Wails requests the exit from the main thread's message loop; if that
	// never completes (a stuck window procedure, a modal loop, ...) the user
	// would be left with an app that cannot be closed at all, so arm a fallback
	// that terminates the process even when the graceful path stalls.
	t.forceExitAfter(quitFallbackTimeout)

	if ctx := t.app.GetCtx(); ctx != nil {
		runtime.Quit(ctx)
		return
	}
	systray.Quit()
}

// forceExitAfter terminates the process when the graceful quit did not finish
// within d. Nothing is running at this point (quit is blocked otherwise), so
// only the process-level teardown (control server auth file removal) is lost.
func (t *TrayManager) forceExitAfter(d time.Duration) {
	go func() {
		time.Sleep(d)
		t.logSystem("Graceful quit did not finish within %s, forcing process exit", d)
		os.Exit(0)
	}()
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
	systray.Quit()
}
