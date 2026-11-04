//go:build windows

package updater

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// installPlatform stages the new executable and starts a detached batch script.
// Windows allows renaming a running .exe but not deleting it, so the script
// waits for this process to exit, renames the old file to .old, activates the
// new one and relaunches.
func installPlatform(ctx context.Context, asset Asset, downloaded string, opts InstallOptions) (InstallResult, error) {
	if asset.Kind != "exe" {
		return InstallResult{}, fmt.Errorf("%w: %s package, use the download page", ErrUnsupported, asset.Kind)
	}
	target := opts.Target
	if target == "" {
		exe, err := os.Executable()
		if err != nil {
			return InstallResult{}, err
		}
		target, err = filepath.Abs(exe)
		if err != nil {
			return InstallResult{}, err
		}
	}
	if err := ensureDir(target); err != nil {
		return InstallResult{}, err
	}

	next := target + ".new"
	logf(opts, "staging %s", next)
	if err := copyFile(downloaded, next, 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("write new executable: %w", err)
	}

	script := filepath.Join(os.TempDir(), fmt.Sprintf("prochub-update-%d.bat", os.Getpid()))
	body := "@echo off\r\n" +
		"setlocal\r\n" +
		":wait\r\n" +
		"tasklist /FI \"PID eq " + fmt.Sprint(os.Getpid()) + "\" 2>NUL | find \"" + fmt.Sprint(os.Getpid()) + "\" >NUL\r\n" +
		"if not errorlevel 1 (\r\n" +
		"  timeout /t 1 /nobreak >NUL\r\n" +
		"  goto wait\r\n" +
		")\r\n" +
		"move /Y \"" + target + "\" \"" + target + ".old\" >NUL\r\n" +
		"if errorlevel 1 goto fail\r\n" +
		"move /Y \"" + next + "\" \"" + target + "\" >NUL\r\n" +
		"if errorlevel 1 goto rollback\r\n" +
		"start \"\" \"" + target + "\"\r\n" +
		"del \"%~f0\"\r\n" +
		"exit /b 0\r\n" +
		":rollback\r\n" +
		"move /Y \"" + target + ".old\" \"" + target + "\" >NUL\r\n" +
		":fail\r\n" +
		"exit /b 1\r\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		return InstallResult{}, err
	}

	cmd := exec.Command("cmd", "/c", script)
	if err := cmd.Start(); err != nil {
		os.Remove(script)
		return InstallResult{}, fmt.Errorf("start update helper: %w", err)
	}
	_ = cmd.Process.Release()

	return InstallResult{
		Message:         "the application will restart to finish the update",
		RestartRequired: true,
	}, nil
}
