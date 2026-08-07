//go:build windows

package platform

import (
	"os"
	"path/filepath"
)

// homeDirFallback returns the user home directory from Windows environment
// variables: USERPROFILE first, then HOMEDRIVE + HOMEPATH.
func homeDirFallback() string {
	if home := os.Getenv("USERPROFILE"); home != "" {
		return home
	}
	drive := os.Getenv("HOMEDRIVE")
	homePath := os.Getenv("HOMEPATH")
	if drive != "" && homePath != "" {
		return filepath.Join(drive, homePath)
	}
	return ""
}
