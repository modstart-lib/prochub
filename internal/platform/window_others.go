//go:build !windows

package platform

// restoreMainWindowNative is a no-op on non-Windows platforms. Wails'
// WindowUnminimise/WindowShow already restore hidden windows on macOS and Linux.
func restoreMainWindowNative() {}

// DescribeMainWindow is a no-op on non-Windows platforms, where no separate
// native window lookup is needed.
func DescribeMainWindow() string { return "" }

// PostQuitToMainWindow is only needed on Windows, where the exit has to reach
// the main window's message loop. Elsewhere Wails' own quit already ends the
// event loop directly.
func PostQuitToMainWindow() (posted bool, detail string) {
	return false, "not supported on this platform"
}
