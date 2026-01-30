package main

import (
	"context"
	"embed"
	goruntime "runtime"

	"fyne.io/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

//go:embed build/trayicon.png
var trayIcon []byte

// Global app reference for tray menu callbacks
var globalApp *App
var trayManager *TrayManager

// createApplicationMenu creates the application menu with Edit menu
// Returns nil on Windows to hide the menu bar
func createApplicationMenu() *menu.Menu {
	// On Windows, return nil to hide menu bar
	// Keyboard shortcuts (Ctrl+C/V/X/A/Z) work natively via WebviewGpuIsDisabled=false
	if goruntime.GOOS == "windows" {
		return nil
	}

	appMenu := menu.NewMenu()

	if goruntime.GOOS == "darwin" {
		// App menu (macOS only)
		appMenu.Append(menu.AppMenu())
		// Edit menu with standard shortcuts using roles (macOS)
		appMenu.Append(menu.EditMenu())
	} else {
		// Edit menu with standard shortcuts (Linux)
		editMenu := appMenu.AddSubmenu("Edit")
		editMenu.AddText("Undo", keys.CmdOrCtrl("z"), func(_ *menu.CallbackData) {})
		editMenu.AddText("Redo", keys.CmdOrCtrl("shift+z"), func(_ *menu.CallbackData) {})
		editMenu.AddSeparator()
		editMenu.AddText("Cut", keys.CmdOrCtrl("x"), func(_ *menu.CallbackData) {})
		editMenu.AddText("Copy", keys.CmdOrCtrl("c"), func(_ *menu.CallbackData) {})
		editMenu.AddText("Paste", keys.CmdOrCtrl("v"), func(_ *menu.CallbackData) {})
		editMenu.AddText("Select All", keys.CmdOrCtrl("a"), func(_ *menu.CallbackData) {})
	}

	return appMenu
}

// wrapStartup wraps the app startup to initialize the tray after Wails context is ready
func wrapStartup(app *App) func(ctx context.Context) {
	return func(ctx context.Context) {
		// Call original startup
		app.startup(ctx)

		// Ensure window is shown on startup
		runtime.WindowShow(ctx)

		// Initialize and start system tray after Wails is ready
		trayManager = NewTrayManager(app)
		go trayManager.Run()
	}
}

// wrapShutdown wraps the app shutdown to cleanup the tray
func wrapShutdown(app *App) func(ctx context.Context) {
	return func(ctx context.Context) {
		// Stop systray
		if trayManager != nil {
			systray.Quit()
		}
		// Call original shutdown
		app.shutdown(ctx)
	}
}

func main() {
	// Create an instance of the app structure
	app := NewApp()
	globalApp = app

	// Create application with options
	err := wails.Run(&options.App{
		Title:         "Prochub",
		Width:         1024,
		Height:        720,
		MinWidth:      900,
		MinHeight:     640,
		Frameless:     false,
		DisableResize: false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:         &options.RGBA{R: 244, G: 239, B: 232, A: 1},
		EnableDefaultContextMenu: false,
		Menu:                     createApplicationMenu(),
		CSSDragValue:             "drag",
		CSSDragProperty:          "--wails-draggable",

		// Custom close behavior: hide window and Dock icon instead of quitting
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			// Hide the window
			runtime.WindowHide(ctx)
			// Hide Dock icon on macOS
			HideDockIcon()
			// Prevent the default close behavior (quitting the app)
			return true
		},

		OnStartup:  wrapStartup(app),
		OnShutdown: wrapShutdown(app),
		Bind: []interface{}{
			app,
		},

		// macOS specific options
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: false,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			About: &mac.AboutInfo{
				Title:   "ProcHub",
				Message: "Process Manager",
				Icon:    icon,
			},
		},

		// Windows specific options
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},

		// Linux specific options
		Linux: &linux.Options{
			Icon:                icon,
			WindowIsTranslucent: false,
		},

		// Single instance lock - show window when second instance is launched
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "prochub-app-unique-id",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				if globalApp != nil && globalApp.ctx != nil {
					// Show Dock icon first (macOS)
					ShowDockIcon()
					// Show the window
					runtime.WindowShow(globalApp.ctx)
					if goruntime.GOOS == "darwin" {
						runtime.WindowSetAlwaysOnTop(globalApp.ctx, true)
						runtime.WindowSetAlwaysOnTop(globalApp.ctx, false)
					}
				}
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
