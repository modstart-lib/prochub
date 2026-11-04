//go:build darwin

package updater

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// installPlatform unpacks the new .app next to the old one and starts a detached
// helper that waits for this process to exit, replaces the bundle and reopens it.
func installPlatform(ctx context.Context, asset Asset, downloaded string, opts InstallOptions) (InstallResult, error) {
	if asset.Kind != "zip" {
		return InstallResult{}, fmt.Errorf("%w: %s package, open the dmg or the download page", ErrUnsupported, asset.Kind)
	}
	target := opts.Target
	if target == "" {
		var err error
		target, err = currentAppBundle()
		if err != nil {
			return InstallResult{}, err
		}
	}
	if !strings.HasSuffix(target, ".app") {
		return InstallResult{}, fmt.Errorf("%w: not running from an .app bundle", ErrUnsupported)
	}

	tmpDir, err := os.MkdirTemp("", "prochub-update-")
	if err != nil {
		return InstallResult{}, err
	}
	logf(opts, "unpacking %s", filepath.Base(downloaded))
	newApp, err := unzipApp(downloaded, tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return InstallResult{}, fmt.Errorf("unpack update: %w", err)
	}

	if _, err := exec.LookPath("codesign"); err == nil {
		out, err := exec.Command("codesign", "--verify", "--deep", "--strict", newApp).CombinedOutput()
		if err != nil {
			os.RemoveAll(tmpDir)
			return InstallResult{}, fmt.Errorf("codesign verify failed: %v: %s", err, strings.TrimSpace(string(out)))
		}
	}

	script := filepath.Join(tmpDir, "apply.sh")
	content := "#!/bin/sh\n" +
		"set -e\n" +
		"while kill -0 " + strconv.Itoa(os.Getpid()) + " 2>/dev/null; do sleep 0.2; done\n" +
		"rm -rf " + shellQuote(target) + "\n" +
		"cp -R " + shellQuote(newApp) + " " + shellQuote(target) + "\n" +
		"xattr -dr com.apple.quarantine " + shellQuote(target) + " 2>/dev/null || true\n" +
		"open " + shellQuote(target) + "\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		os.RemoveAll(tmpDir)
		return InstallResult{}, err
	}

	cmd := exec.Command("/bin/sh", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		os.RemoveAll(tmpDir)
		return InstallResult{}, fmt.Errorf("start update helper: %w", err)
	}
	_ = cmd.Process.Release()

	return InstallResult{
		Message:         "the application will restart to finish the update",
		RestartRequired: true,
	}, nil
}

func currentAppBundle() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// .../ProcHub.app/Contents/MacOS/ProcHub
	app := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if strings.HasSuffix(app, ".app") {
		return app, nil
	}
	return "", fmt.Errorf("%w: cannot locate .app bundle from %s", ErrUnsupported, exe)
}

func unzipApp(zipPath, destDir string) (string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return "", err
	}
	var appRoot string
	for _, file := range reader.File {
		name := file.Name
		if strings.HasPrefix(name, "__MACOSX/") {
			continue
		}
		if appRoot == "" {
			if i := strings.Index(name, ".app/"); i > 0 {
				appRoot = name[:i+4]
			}
		}
		target := filepath.Join(destAbs, name)
		if !strings.HasPrefix(target, destAbs+string(os.PathSeparator)) {
			return "", fmt.Errorf("unsafe zip entry %q", name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return "", err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		src, err := file.Open()
		if err != nil {
			return "", err
		}
		mode := file.Mode()
		if mode&os.ModeSymlink != 0 {
			link, err := io.ReadAll(src)
			src.Close()
			if err != nil {
				return "", err
			}
			if err := os.Symlink(string(link), target); err != nil {
				return "", err
			}
			continue
		}
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
		if err != nil {
			src.Close()
			return "", err
		}
		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			return "", copyErr
		}
	}
	if appRoot == "" {
		return "", fmt.Errorf("zip contains no .app bundle")
	}
	return filepath.Join(destAbs, appRoot), nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
