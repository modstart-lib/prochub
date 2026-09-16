//go:build !windows

package platform

// restoreMainWindowNative is a no-op on non-Windows platforms. Wails'
// WindowUnminimise/WindowShow already restore hidden windows on macOS and Linux.
func restoreMainWindowNative() {}
