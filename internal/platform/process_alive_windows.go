//go:build windows

package platform

import (
	"errors"

	"golang.org/x/sys/windows"
)

// ProcessAlive reports whether a process with the given PID currently exists.
// A PID that cannot be opened because of an access denial is treated as alive:
// an instance running under a different integrity level still exists, and
// reporting it dead would let a duplicate launch start.
func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return errors.Is(err, windows.ERROR_ACCESS_DENIED)
	}
	_ = windows.CloseHandle(handle)
	return true
}
