package main

import (
	"fmt"
	"os"
	"time"

	"prochub/internal/control"
	"prochub/internal/logging"
	"prochub/internal/platform"

	"github.com/wailsapp/wails/v2/pkg/options"
)

// runningInstanceTimeout bounds how long the startup probe waits for a running
// instance to answer, so a wedged first instance cannot delay a new launch.
const runningInstanceTimeout = 3 * time.Second

// handleExistingInstance checks whether another ProcHub instance is already
// serving the control endpoint for dataDir. When one is found it asks that
// instance to show its window and returns true, so the caller exits before
// Wails (and its WebView2 environment, which cannot share a user data folder
// with a running instance) starts a second process.
//
// Every failure path returns false: a missing or stale auth file, a PID that was
// reused by an unrelated process and an instance that cannot be reached must
// never block a launch.
func handleExistingInstance(dataDir string, logger *logging.RollingStore) bool {
	info, err := control.ReadAuth(dataDir)
	if err != nil {
		if !os.IsNotExist(err) {
			logSingleInstance(logger, fmt.Sprintf("cannot read auth.json (%v), starting a new instance", err))
		}
		return false
	}

	if !platform.ProcessAlive(info.PID) {
		logSingleInstance(logger, fmt.Sprintf("stale auth.json points to dead pid=%d, starting a new instance", info.PID))
		return false
	}

	client, err := control.NewClientWithTimeout(dataDir, runningInstanceTimeout)
	if err != nil {
		return false
	}

	status, err := client.Status()
	if err != nil {
		logSingleInstance(logger, fmt.Sprintf("auth.json pid=%d is unreachable (%v), starting a new instance", info.PID, err))
		return false
	}
	if status.PID != info.PID {
		logSingleInstance(logger, fmt.Sprintf("auth.json pid=%d does not match control server pid=%d, starting a new instance", info.PID, status.PID))
		return false
	}

	if err := client.Show(); err != nil {
		logSingleInstance(logger, fmt.Sprintf("failed to ask running instance pid=%d to show its window: %v", info.PID, err))
		return false
	}

	logSingleInstance(logger, fmt.Sprintf("another instance is already running (pid=%d), window shown, exiting", info.PID))
	return true
}

// logSecondInstanceLaunch records that Wails' single-instance lock intercepted
// a second process. That lock is the fallback taken while the running instance
// is still starting, or when the startup probe could not reach it.
func logSecondInstanceLaunch(data options.SecondInstanceData) {
	if globalApp == nil {
		return
	}
	globalApp.LogSystemError("single-instance", fmt.Sprintf(
		"second instance intercepted by the Wails lock (args=%v, cwd=%s)", data.Args, data.WorkingDirectory))
}

// logSingleInstance appends one line to the application system log. A nil
// logger is ignored so bootstrap code can run without one.
func logSingleInstance(logger *logging.RollingStore, message string) {
	if logger == nil {
		return
	}
	_ = logger.Append(logging.Entry{
		Timestamp: time.Now(),
		Stream:    "single-instance",
		Line:      message,
	})
}
