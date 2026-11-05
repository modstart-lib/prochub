//go:build linux

package updater

import (
	"context"
	"fmt"
	"os"
)

func init() {
	installDependencyCheck = checkAppImageDependencies
}

// installPlatform replaces a running AppImage in place. A .deb installed by the
// system package manager is not touched: it needs root and dpkg/apt state.
func installPlatform(ctx context.Context, asset Asset, downloaded string, opts InstallOptions) (InstallResult, error) {
	if asset.Kind != "AppImage" {
		return InstallResult{}, fmt.Errorf("%w: %s package, use the package manager or the download page", ErrUnsupported, asset.Kind)
	}
	target := opts.Target
	if target == "" {
		target = os.Getenv("APPIMAGE")
	}
	if target == "" {
		return InstallResult{}, fmt.Errorf("%w: not running as an AppImage", ErrUnsupported)
	}
	if err := ensureDir(target); err != nil {
		return InstallResult{}, err
	}
	// Never replace a working AppImage with one whose system libraries are
	// missing; that would leave the user unable to start the app.
	if installDependencyCheck != nil {
		if err := installDependencyCheck(downloaded); err != nil {
			return InstallResult{}, err
		}
	}

	logf(opts, "replacing %s", target)
	next := target + ".new"
	if err := copyFile(downloaded, next, 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("write new AppImage: %w", err)
	}
	backup := target + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil {
		os.Remove(next)
		return InstallResult{}, fmt.Errorf("back up current AppImage: %w", err)
	}
	if err := os.Rename(next, target); err != nil {
		_ = os.Rename(backup, target)
		os.Remove(next)
		return InstallResult{}, fmt.Errorf("activate new AppImage: %w", err)
	}
	if err := os.Chmod(target, 0o755); err != nil {
		return InstallResult{}, err
	}
	return InstallResult{
		Message:         "AppImage replaced; restart ProcHub to use the new version",
		RestartRequired: true,
	}, nil
}
