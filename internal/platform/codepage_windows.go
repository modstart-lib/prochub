//go:build windows

package platform

import "golang.org/x/sys/windows"

// activeCodePage returns the text code page that console programs on this
// machine use for redirected output. It mirrors the .NET runtime: prefer the
// attached console's output code page, and fall back to the system ANSI code
// page (CP936/GBK on Simplified Chinese Windows) when there is no console, as
// is the case for a GUI host like ProcHub.
func activeCodePage() uint32 {
	if cp, err := windows.GetConsoleOutputCP(); err == nil && cp != 0 {
		return cp
	}
	return windows.GetACP()
}
