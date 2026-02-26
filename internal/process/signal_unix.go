//go:build !windows

package process

import (
	"os"
	"os/exec"
	"syscall"
)

// setupProcessGroup sets up process group for Unix-like systems
// This allows us to kill the entire process tree
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// killProcess kills a process and its children on Unix-like systems
func killProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// Try graceful termination first with SIGTERM
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		// Kill the entire process group
		if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
			// If SIGTERM fails, try SIGKILL
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		}
	} else {
		// Fallback to killing just the process
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return cmd.Process.Kill()
		}
	}

	return nil
}

// gracefulStop attempts to gracefully stop a process with SIGTERM
// and waits for the specified duration before force killing
func gracefulStop(cmd *exec.Cmd, done chan struct{}) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}

	// Signal that we've sent SIGTERM
	close(done)
	return nil
}

// isProcessRunning checks if a process is still running
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds, so we need to send signal 0
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// getExitCode returns the exit code of a finished process
func getExitCode(cmd *exec.Cmd) int {
	if cmd == nil || cmd.ProcessState == nil {
		return -1
	}
	return cmd.ProcessState.ExitCode()
}
