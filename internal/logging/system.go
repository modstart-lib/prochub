package logging

import (
	"os"
	"path/filepath"
)

const (
	// systemLogDirName is the subdirectory of the data directory that holds the
	// application system log.
	systemLogDirName = "system_logs"
	// systemLogMaxLines and systemLogMaxFiles configure log rotation.
	systemLogMaxLines = 1000
	systemLogMaxFiles = 10
)

// SystemLogDir returns the system log directory inside dataDir without creating
// it.
func SystemLogDir(dataDir string) string {
	return filepath.Join(dataDir, systemLogDirName)
}

// NewSystemLogger returns the rolling store backing the application system log
// and creates the log directory when it is missing. It is safe to use before
// the application configuration is loaded, so early bootstrap code (the process
// single-instance guard) can log alongside the running application.
func NewSystemLogger(dataDir string) *RollingStore {
	dir := SystemLogDir(dataDir)
	_ = os.MkdirAll(dir, 0o755)
	return NewRollingStore(dir, systemLogMaxLines, systemLogMaxFiles)
}
