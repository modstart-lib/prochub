package platform

import (
	"os"
	"path/filepath"
)

// appDirName is the application data directory name under the user home.
// The same directory (~/.prochub) is used on every OS so configuration and
// logs stay in one place.
const appDirName = ".prochub"

// DataDir returns the application data directory (e.g. ~/.prochub) and
// creates it if it does not exist yet.
func DataDir() (string, error) {
	home, err := homeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, appDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// MustDataDir is like DataDir but never returns an error: it falls back to a
// directory under the OS temp dir when the user home cannot be resolved.
func MustDataDir() string {
	dir, err := DataDir()
	if err != nil {
		return filepath.Join(os.TempDir(), appDirName)
	}
	return dir
}

// homeDir resolves the user home directory using the standard OS lookup,
// falling back to platform-specific environment variables.
func homeDir() (string, error) {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home, nil
	}
	if home := homeDirFallback(); home != "" {
		return home, nil
	}
	return "", os.ErrNotExist
}
