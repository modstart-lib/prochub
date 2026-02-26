package service

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

var (
	ErrUnsupportedPlatform = errors.New("unsupported platform")
	ErrAppPathNotFound     = errors.New("application path not found")
)

// AutoStartManager manages auto-start functionality across platforms
type AutoStartManager struct {
	appName     string
	displayName string
	appPath     string
}

// NewAutoStartManager creates a new auto-start manager
func NewAutoStartManager(appName, displayName string) *AutoStartManager {
	return &AutoStartManager{
		appName:     appName,
		displayName: displayName,
		appPath:     getExecutablePath(),
	}
}

// IsEnabled checks if auto-start is currently enabled
func (m *AutoStartManager) IsEnabled() (bool, error) {
	return isAutoStartEnabled(m.appName, m.appPath)
}

// Enable enables auto-start for the application
func (m *AutoStartManager) Enable() error {
	if m.appPath == "" {
		return ErrAppPathNotFound
	}
	return enableAutoStart(m.appName, m.displayName, m.appPath)
}

// Disable disables auto-start for the application
func (m *AutoStartManager) Disable() error {
	return disableAutoStart(m.appName)
}

// GetPlatform returns the current platform
func (m *AutoStartManager) GetPlatform() string {
	return runtime.GOOS
}

// getExecutablePath returns the path to the current executable
func getExecutablePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	// Resolve symlinks
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return ""
	}
	return exe
}
