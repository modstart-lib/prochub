//go:build windows

package service

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

// wslCommandTimeout bounds every wsl.exe invocation so a stuck WSL service
// never blocks the UI.
const wslCommandTimeout = 30 * time.Second

// wslBootTimeout bounds how long a boot is awaited before giving up. WSL cold
// boots can take tens of seconds because the login shell runs the profile.
const wslBootTimeout = 120 * time.Second

// hiddenSysProcAttr launches console programs without ever creating a visible
// console window. ProcHub is a GUI application, so a bare exec.Command would
// otherwise flash a black console window on every wsl.exe call.
func hiddenSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}

// keepAliveCmd references the detached background process used by wslStart's
// fallback so its Wait call can release the process handle once it exits.
var (
	keepAliveMu  sync.Mutex
	keepAliveCmd *exec.Cmd
)

// wslSupported reports whether the wsl.exe launcher exists on this machine.
func wslSupported() bool {
	if _, err := exec.LookPath("wsl.exe"); err == nil {
		return true
	}
	_, err := exec.LookPath("wsl")
	return err == nil
}

// wslStatus reads the installed distributions via `wsl.exe -l -v`.
func wslStatus() WSLStatus {
	status := WSLStatus{}
	if !wslSupported() {
		return status
	}
	status.Supported = true

	out, err := runWSL("-l", "-v")
	if err != nil && strings.TrimSpace(out) == "" {
		return status
	}

	distros := parseWSLDistros(out)
	status.Available = len(distros) > 0
	status.Distros = distros
	for _, distro := range distros {
		if distro.Default {
			status.DefaultDistro = distro.Name
			status.Version = distro.Version
		}
		switch strings.ToLower(distro.State) {
		case "running":
			status.Running = true
		case "installing", "starting", "converting":
			status.Starting = true
		}
	}

	return status
}

// parseWSLDistros parses the decoded output of `wsl.exe -l -v`.
func parseWSLDistros(out string) []WSLDistro {
	var distros []WSLDistro
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}

		trimmed := strings.TrimSpace(line)
		isDefault := strings.HasPrefix(trimmed, "*")
		if isDefault {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "*"))
		}

		fields := strings.Fields(trimmed)
		// A distribution row looks like: NAME STATE VERSION.
		if len(fields) < 3 || strings.EqualFold(fields[0], "NAME") {
			continue
		}

		version := fields[len(fields)-1]
		state := fields[len(fields)-2]
		name := strings.Join(fields[:len(fields)-2], " ")

		distros = append(distros, WSLDistro{
			Name:    name,
			State:   state,
			Version: version,
			Default: isDefault,
		})
	}
	return distros
}

// wslStart boots the default distribution the same way the `wsl` command does
// in a terminal: through a login (and interactive) shell. That makes
// /etc/profile, ~/.profile and ~/.bashrc run, so services and the environment
// match a normal terminal session. The shell is held open by a long-running
// command, which also stops the WSL instance from idling out.
func wslStart() error {
	if !wslSupported() {
		return ErrWSLUnsupported
	}

	status := wslStatus()
	if !status.Available || len(status.Distros) == 0 {
		return ErrWSLNoDistribution
	}

	if status.Running {
		// Already up: attach our login-shell keep-alive so the environment
		// matches a terminal session (a distro booted non-login, e.g. via
		// dbus-launch, would otherwise miss the profile scripts), and so the
		// instance stays alive when other sessions close.
		if !hasKeepAlive() {
			_ = wslSpawnLoginShell(status.DefaultDistro)
		}
		return nil
	}

	distro := status.DefaultDistro
	if err := wslSpawnLoginShell(distro); err != nil {
		// Fallback: boot with dbus-launch when the login shell cannot start.
		if _, dbusErr := runWSL(bootArgs(distro, "dbus-launch", "true")...); dbusErr != nil {
			return err
		}
	}

	// The instance boots asynchronously, so keep the operation "in progress"
	// until the distribution is actually reported as running.
	return waitForWSLRunning(wslBootTimeout)
}

// hasKeepAlive reports whether a keep-alive process is currently held.
func hasKeepAlive() bool {
	keepAliveMu.Lock()
	defer keepAliveMu.Unlock()
	return keepAliveCmd != nil
}

// waitForWSLRunning polls the distribution state until it is running or the
// timeout elapses.
func waitForWSLRunning(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if wslStatus().Running {
			return nil
		}
		if time.Now().After(deadline) {
			return ErrWSLStartTimeout
		}
		time.Sleep(time.Second)
	}
}

// bootArgs builds the argument list for booting a distribution, targeting the
// default distribution when its name is known.
func bootArgs(distro string, command ...string) []string {
	args := []string{"--exec"}
	args = append(args, command...)
	if distro != "" {
		args = append([]string{"-d", distro}, args...)
	}
	return args
}

// wslSpawnLoginShell starts a detached, hidden login+interactive shell inside
// the distribution and holds it open with `sleep infinity`. Holding a process
// under the WSL init keeps the instance alive, while the login shell mirrors
// what `wsl` does in a terminal (profile scripts, user login session, dbus).
func wslSpawnLoginShell(distro string) error {
	args := []string{"--exec", "/bin/bash", "-l", "-i", "-c", "exec sleep infinity"}
	if distro != "" {
		args = append([]string{"-d", distro}, args...)
	}

	cmd := exec.Command("wsl.exe", args...)
	cmd.SysProcAttr = hiddenSysProcAttr()
	// stdin/stdout/stderr fall back to the null device; `exec sleep infinity`
	// keeps the shell alive without needing any pipes.
	if err := cmd.Start(); err != nil {
		return err
	}

	keepAliveMu.Lock()
	keepAliveCmd = cmd
	keepAliveMu.Unlock()

	go func() {
		_ = cmd.Wait()
		keepAliveMu.Lock()
		if keepAliveCmd == cmd {
			keepAliveCmd = nil
		}
		keepAliveMu.Unlock()
	}()

	return nil
}

// wslRestart shuts the whole WSL subsystem down and boots it again.
func wslRestart() error {
	if !wslSupported() {
		return ErrWSLUnsupported
	}

	if _, err := runWSL("--shutdown"); err != nil {
		return err
	}

	// Give the subsystem a brief moment to release the VM before booting it.
	time.Sleep(500 * time.Millisecond)
	return wslStart()
}

// runWSL executes wsl.exe with the given arguments and returns its combined
// output decoded from the encoding wsl.exe uses when its output is piped.
func runWSL(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), wslCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wsl.exe", args...)
	cmd.SysProcAttr = hiddenSysProcAttr()
	// Ask WSL for UTF-8 output when supported; older versions keep emitting
	// UTF-16LE, which decodeUTF16LE still handles.
	cmd.Env = append(os.Environ(), "WSL_UTF8=1")

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	return decodeUTF16LE(buf.Bytes()), err
}

// decodeUTF16LE decodes wsl.exe output, which is UTF-16LE when redirected,
// while falling back to plain UTF-8 for ASCII-only payloads.
func decodeUTF16LE(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	hasBOM := len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE
	if !hasBOM {
		// Heuristic: UTF-16LE ASCII text has a NUL in every high byte.
		nuls := 0
		for i := 1; i < len(b); i += 2 {
			if b[i] == 0 {
				nuls++
			}
		}
		if nuls <= len(b)/8 {
			return string(b)
		}
	} else {
		b = b[2:]
	}

	u16 := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u16 = append(u16, uint16(b[i])|uint16(b[i+1])<<8)
	}
	return string(utf16.Decode(u16))
}
