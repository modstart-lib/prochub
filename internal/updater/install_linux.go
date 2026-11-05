//go:build linux

package updater

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// relaunchAfterInstall starts the replaced AppImage once the old process exits.
// Tests switch it off because they do not run the real application lifecycle.
var relaunchAfterInstall = true

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
	if err := scheduleRelaunch(target); err != nil {
		// The replacement already succeeded; a failed relaunch only means the
		// user has to start ProcHub manually.
		logf(opts, "cannot schedule relaunch: %v", err)
	}
	return InstallResult{
		Message:         "AppImage replaced; ProcHub is restarting with the new version",
		RestartRequired: true,
	}, nil
}

// scheduleRelaunch starts a detached helper that waits for this process to
// exit and then opens the replaced AppImage. Starting it earlier would race
// with the AppImage mount that is still in use.
func scheduleRelaunch(target string) error {
	if !relaunchAfterInstall {
		return nil
	}
	script := filepath.Join(os.TempDir(), fmt.Sprintf("prochub-update-%d.sh", os.Getpid()))
	quoted := "'" + strings.ReplaceAll(target, "'", `'\''`) + "'"
	content := "#!/bin/sh\n" +
		"while kill -0 " + strconv.Itoa(os.Getpid()) + " 2>/dev/null; do sleep 0.2; done\n" +
		"rm -f \"$0\"\n" +
		"nohup " + quoted + " >/dev/null 2>&1 &\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("/bin/sh", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		os.Remove(script)
		return err
	}
	_ = cmd.Process.Release()
	// Give the helper a moment to start before the caller quits the app.
	time.Sleep(50 * time.Millisecond)
	return nil
}
