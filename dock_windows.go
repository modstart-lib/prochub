//go:build windows

package main

// HideDockIcon is a no-op on Windows (Windows doesn't have a Dock)
// The taskbar icon is handled by the window visibility
func HideDockIcon() {
	// No-op on Windows
}

// ShowDockIcon is a no-op on Windows
func ShowDockIcon() {
	// No-op on Windows
}
