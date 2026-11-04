package updater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	// ErrUnsupported means this distribution format cannot be replaced in place
	// (for example a .deb installed by the system package manager).
	ErrUnsupported = errors.New("automatic install is not supported for this package")
	// ErrProcessesRunning blocks an install while managed processes still run.
	ErrProcessesRunning = errors.New("stop all processes before updating")
)

// InstallOptions controls the platform install step.
type InstallOptions struct {
	// Target overrides the path that will be replaced. Used by tests and by
	// wrapper distributions; empty means the platform default.
	Target string
	// Logf receives progress messages from the installer.
	Logf func(format string, args ...interface{})
}

// Install replaces the current installation with the downloaded asset.
func Install(ctx context.Context, asset Asset, downloaded string, opts InstallOptions) (InstallResult, error) {
	if downloaded == "" {
		return InstallResult{}, fmt.Errorf("no downloaded file")
	}
	if _, err := os.Stat(downloaded); err != nil {
		return InstallResult{}, err
	}
	return installPlatform(ctx, asset, downloaded, opts)
}

func logf(opts InstallOptions, format string, args ...interface{}) {
	if opts.Logf != nil {
		opts.Logf(format, args...)
	}
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	// Windows does not replace an existing destination with Rename.
	_ = os.Remove(dst)
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
