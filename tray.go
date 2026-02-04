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
	app   *App
	mShow *systray.MenuItem
	mQuit *systray.MenuItem
}

// NewTrayManager creates a new TrayManager
func NewTrayManager(app *App) *TrayManager {
	return &TrayManager{
		app: app,
	}
}

// getLocalizedLabels returns menu labels based on app locale setting
func (t *TrayManager) getLocalizedLabels() (showLabel, quitLabel string) {
	showLabel = "Show Window"
	quitLabel = "Quit"

	// Check app configuration first
	if t.app != nil {
		if t.app.config.Locale == "zh" {
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

	// Get labels based on app configuration
	showLabel, quitLabel := t.getLocalizedLabels()

	// Create menu items with localized labels
	t.mShow = systray.AddMenuItem(showLabel, "Show the main window")
	systray.AddSeparator()
	t.mQuit = systray.AddMenuItem(quitLabel, "Quit the application")

	// Handle menu clicks in a goroutine
	go func() {
		for {
			select {
			case <-t.mShow.ClickedCh:
				t.showWindow()
			case <-t.mQuit.ClickedCh:
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
