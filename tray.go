package main

import (
	"os"
	goruntime "runtime"
	"strings"

	"fyne.io/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TrayManager manages the system tray icon and menu
type TrayManager struct {
	app *App
}

// NewTrayManager creates a new TrayManager
func NewTrayManager(app *App) *TrayManager {
	return &TrayManager{
		app: app,
	}
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

// onReady is called when systray is ready
func (t *TrayManager) onReady() {
	// Use different tray icons for different platforms
	// macOS: Black template icon (transparent background)
	// Windows: Colored ICO format icon
	if goruntime.GOOS == "windows" {
		systray.SetIcon(trayIconWindows)
	} else {
		systray.SetIcon(trayIcon)
	}
	systray.SetTooltip("ProcHub - Process Manager")

	// Create menu items with localized labels
	showLabel := "Show Window"
	quitLabel := "Quit"

	// Use Chinese labels if the system locale is Chinese
	if isChineseLocale() {
		showLabel = "显示界面"
		quitLabel = "退出"
	}

	mShow := systray.AddMenuItem(showLabel, "Show the main window")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem(quitLabel, "Quit the application")

	// Handle menu clicks in a goroutine
	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				t.showWindow()
			case <-mQuit.ClickedCh:
				t.quitApp()
				return
			}
		}
	}()
}

// onExit is called when systray is about to exit
func (t *TrayManager) onExit() {
	// Cleanup if needed
}

// showWindow shows the main application window and Dock icon
func (t *TrayManager) showWindow() {
	if t.app != nil && t.app.ctx != nil {
		// Show Dock icon first (macOS)
		ShowDockIcon()
		// Then show the window
		runtime.WindowShow(t.app.ctx)
		// On macOS, bring window to front
		if goruntime.GOOS == "darwin" {
			runtime.WindowSetAlwaysOnTop(t.app.ctx, true)
			runtime.WindowSetAlwaysOnTop(t.app.ctx, false)
		}
	}
}

// quitApp properly quits the application
func (t *TrayManager) quitApp() {
	if t.app != nil && t.app.ctx != nil {
		runtime.Quit(t.app.ctx)
	}
	systray.Quit()
}

// Run starts the systray - should be called in a goroutine
func (t *TrayManager) Run() {
	systray.Run(t.onReady, t.onExit)
}

// Quit stops the systray
func (t *TrayManager) Quit() {
	systray.Quit()
}
