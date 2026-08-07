//go:build !windows

package platform

import "os"

// homeDirFallback returns the user home directory from $HOME on Unix-like
// systems (macOS / Linux).
func homeDirFallback() string {
	return os.Getenv("HOME")
}
