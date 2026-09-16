package platform

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// swRestore shows a window and restores it to its original size and
	// position, bringing back a window hidden with SW_HIDE.
	swRestore = 9
	// mainWindowClass is the window class Wails registers for the main window
	// (see wails NewWindow: windowClassName = "wailsWindow").
	mainWindowClass = "wailsWindow"
	// mainWindowTitle must match options.App.Title in main.go.
	mainWindowTitle = "ProcHub"
)

var (
	user32            = windows.NewLazySystemDLL("user32.dll")
	procFindWindowW   = user32.NewProc("FindWindowW")
	procShowWindow    = user32.NewProc("ShowWindow")
	procSetForeground = user32.NewProc("SetForegroundWindow")
)

// restoreMainWindowNative restores and shows the main window using Win32
// SW_RESTORE. Wails' WindowShow/WindowUnminimise only issue SW_SHOW, which
// cannot bring back a window hidden with SW_HIDE on Windows, so we call
// ShowWindow(SW_RESTORE) on the real HWND here.
func restoreMainWindowNative() {
	class, err := windows.UTF16PtrFromString(mainWindowClass)
	if err != nil {
		return
	}
	title, err := windows.UTF16PtrFromString(mainWindowTitle)
	if err != nil {
		return
	}
	hwnd, _, _ := procFindWindowW.Call(
		uintptr(unsafe.Pointer(class)),
		uintptr(unsafe.Pointer(title)),
	)
	if hwnd == 0 {
		return
	}
	procShowWindow.Call(hwnd, swRestore)
	procSetForeground.Call(hwnd)
}
