//go:build !windows

package platform

import (
	"errors"
	"syscall"
)

// ProcessAlive reports whether a process with the given PID currently exists.
// A PID we are not allowed to signal (EPERM) is treated as alive: an instance
// owned by another user still exists, and reporting it dead would let a
// duplicate launch start.
func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	return errors.Is(err, syscall.EPERM)
}
