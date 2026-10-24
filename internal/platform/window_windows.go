package platform

import (
	"fmt"
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
	// wmQuit asks a thread's message loop to end.
	wmQuit = 0x0012
)

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procFindWindowW              = user32.NewProc("FindWindowW")
	procShowWindow               = user32.NewProc("ShowWindow")
	procSetForeground            = user32.NewProc("SetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procPostThreadMessageW       = user32.NewProc("PostThreadMessageW")
)

// findMainWindow returns the HWND of the Wails main window, or 0 when it does
// not exist (yet).
func findMainWindow() uintptr {
	class, err := windows.UTF16PtrFromString(mainWindowClass)
	if err != nil {
		return 0
	}
	title, err := windows.UTF16PtrFromString(mainWindowTitle)
	if err != nil {
		return 0
	}
	hwnd, _, _ := procFindWindowW.Call(
		uintptr(unsafe.Pointer(class)),
		uintptr(unsafe.Pointer(title)),
	)
	return hwnd
}

// restoreMainWindowNative restores and shows the main window using Win32
// SW_RESTORE. Wails' WindowShow/WindowUnminimise only issue SW_SHOW, which
// cannot bring back a window hidden with SW_HIDE on Windows, so we call
// ShowWindow(SW_RESTORE) on the real HWND here.
func restoreMainWindowNative() {
	hwnd := findMainWindow()
	if hwnd == 0 {
		return
	}
	procShowWindow.Call(hwnd, swRestore)
	procSetForeground.Call(hwnd)
}

// DescribeMainWindow reports the main window handle for diagnostics: a quit or a
// window restore that suddenly cannot find the window is worth seeing in the log.
func DescribeMainWindow() string {
	hwnd := findMainWindow()
	if hwnd == 0 {
		return fmt.Sprintf("window %s/%q not found", mainWindowClass, mainWindowTitle)
	}
	return fmt.Sprintf("hwnd=0x%x", hwnd)
}

// PostQuitToMainWindow ends the main window's message loop, which makes Wails
// return from Run and run its shutdown hooks. Wails' own quit hands the exit to
// the window thread through a custom window message; when that hand-off stops
// arriving (observed after the app had been running for days), posting WM_QUIT
// straight to the owning thread's queue ends the loop the same way.
//
// It reports whether the message was posted plus what it found (window handle,
// owning thread, or the failing Win32 step), which is logged so a quit that
// still fails can be traced to the exact step.
func PostQuitToMainWindow() (posted bool, detail string) {
	hwnd := findMainWindow()
	if hwnd == 0 {
		return false, fmt.Sprintf("main window (%s/%q) not found", mainWindowClass, mainWindowTitle)
	}
	threadID, _, idErr := procGetWindowThreadProcessId.Call(hwnd)
	if threadID == 0 {
		return false, fmt.Sprintf("GetWindowThreadProcessId(hwnd=0x%x) failed: %v", hwnd, idErr)
	}
	postOk, _, postErr := procPostThreadMessageW.Call(threadID, wmQuit, 0, 0)
	if postOk == 0 {
		return false, fmt.Sprintf("PostThreadMessage(thread=%d, WM_QUIT) failed: %v", threadID, postErr)
	}
	return true, fmt.Sprintf("hwnd=0x%x thread=%d", hwnd, threadID)
}
