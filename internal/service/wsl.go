package service

import (
	"errors"
	"sync/atomic"
)

var (
	// ErrWSLUnsupported is returned when running on a platform without WSL support.
	ErrWSLUnsupported = errors.New("WSL is only supported on Windows")
	// ErrWSLNotInstalled is returned when the wsl.exe command is unavailable.
	ErrWSLNotInstalled = errors.New("WSL is not installed")
	// ErrWSLNoDistribution is returned when WSL has no distribution installed.
	ErrWSLNoDistribution = errors.New("no WSL distribution installed")
	// ErrWSLStartTimeout is returned when WSL does not come up in time.
	ErrWSLStartTimeout = errors.New("timed out waiting for WSL to start")
)

// WSLDistro describes a single WSL distribution and its runtime state.
type WSLDistro struct {
	Name    string `json:"name"`
	State   string `json:"state"`
	Version string `json:"version"`
	Default bool   `json:"default"`
}

// WSLStatus is a snapshot of the Windows Subsystem for Linux runtime state.
type WSLStatus struct {
	// Supported reports whether the current platform can manage WSL at all.
	Supported bool `json:"supported"`
	// Available reports whether WSL is installed and reachable.
	Available bool `json:"available"`
	// Running reports whether at least one distribution (or the WSL VM) is up.
	Running bool `json:"running"`
	// Starting reports that a boot/reload is currently in progress. WSL can take
	// tens of seconds to cold boot, so the UI shows a "starting" state instead of
	// falling back to "stopped" during that window.
	Starting bool `json:"starting"`
	// Version is the WSL version of the default distribution (1 or 2).
	Version string `json:"version"`
	// DefaultDistro is the name of the default distribution.
	DefaultDistro string `json:"defaultDistro"`
	// Distros lists every installed distribution.
	Distros []WSLDistro `json:"distros"`
}

// WSLManager controls the Windows Subsystem for Linux.
type WSLManager struct {
	// busy is 1 while a start or restart operation is running. It is updated
	// with atomics because Status may be polled from the UI concurrently.
	busy int32
}

// NewWSLManager creates a new WSL manager.
func NewWSLManager() *WSLManager {
	return &WSLManager{}
}

// Status returns the current WSL runtime status.
func (m *WSLManager) Status() WSLStatus {
	status := wslStatus()
	if atomic.LoadInt32(&m.busy) == 1 {
		status.Starting = true
	}
	return status
}

// Start boots the default WSL distribution and keeps it running.
func (m *WSLManager) Start() error {
	atomic.StoreInt32(&m.busy, 1)
	defer atomic.StoreInt32(&m.busy, 0)
	return wslStart()
}

// Restart shuts down the whole WSL subsystem and boots it again.
func (m *WSLManager) Restart() error {
	atomic.StoreInt32(&m.busy, 1)
	defer atomic.StoreInt32(&m.busy, 0)
	return wslRestart()
}
