//go:build !windows

package service

// wslStatus reports WSL as unsupported on non-Windows platforms.
func wslStatus() WSLStatus {
	return WSLStatus{Supported: false}
}

// wslStart is a no-op placeholder on non-Windows platforms.
func wslStart() error {
	return ErrWSLUnsupported
}

// wslRestart is a no-op placeholder on non-Windows platforms.
func wslRestart() error {
	return ErrWSLUnsupported
}
