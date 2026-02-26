//go:build windows

package process

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

// Windows creation flags
const (
	CREATE_NO_WINDOW = 0x08000000
)

// setupProcessGroup sets up process group for Windows
// Creates a new process group for job control and hides the console window
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | CREATE_NO_WINDOW,
		HideWindow:    true,
	}
}

// killProcess kills a process and its children on Windows
func killProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// On Windows, we use taskkill to kill the process tree
	kill := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
	kill.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	err := kill.Run()
	if err != nil {
		// Fallback to direct kill
		return cmd.Process.Kill()
	}
	return nil
}

// gracefulStop attempts to gracefully stop a process on Windows
// Windows doesn't have SIGTERM equivalent, so we send CTRL_BREAK_EVENT
func gracefulStop(cmd *exec.Cmd, done chan struct{}) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// On Windows, we try CTRL_BREAK_EVENT for console apps
	// This may not work for all applications
	dll, err := syscall.LoadDLL("kernel32.dll")
	if err == nil {
		proc, err := dll.FindProc("GenerateConsoleCtrlEvent")
		if err == nil {
			// Send CTRL_BREAK_EVENT (1) to the process group
			_, _, _ = proc.Call(1, uintptr(cmd.Process.Pid))
		}
	}

	// Signal that we've attempted graceful stop
	close(done)
	return nil
}

// isProcessRunning checks if a process is still running on Windows
func isProcessRunning(pid int) bool {
	_, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Windows, FindProcess always succeeds for valid PIDs
	// We need to open the process to check if it exists
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle)

	var exitCode uint32
	err = syscall.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return false
	}
	// STILL_ACTIVE (259) means the process is still running
	return exitCode == 259
}

// getExitCode returns the exit code of a finished process
func getExitCode(cmd *exec.Cmd) int {
	if cmd == nil || cmd.ProcessState == nil {
		return -1
	}
	return cmd.ProcessState.ExitCode()
}
