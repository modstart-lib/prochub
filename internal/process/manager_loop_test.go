package process

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"prochub/internal/platform"
)

// TestMain lets the test binary double as the child process the manager spawns,
// so the run-loop guard can be exercised on every platform without depending on
// a shell or on OS-specific commands.
func TestMain(m *testing.M) {
	if mode := os.Getenv("PROCHUB_TEST_CHILD"); mode != "" {
		os.Exit(runTestChild(mode))
	}
	os.Exit(m.Run())
}

// runTestChild implements the child behaviours used by the tests:
//   - "fail":                exit non-zero immediately
//   - "run":                 record the pid and stay alive
//   - "fail-until-flag":     fail until PROCHUB_TEST_CHILD_FLAG exists, then "run"
//   - "count-fail":          record the pid and exit non-zero
func runTestChild(mode string) int {
	switch mode {
	case "fail-until-flag":
		if _, err := os.Stat(os.Getenv("PROCHUB_TEST_CHILD_FLAG")); err != nil {
			return 1
		}
		fallthrough
	case "run":
		recordPID()
		time.Sleep(30 * time.Second)
		return 0
	case "count-fail":
		recordPID()
		return 1
	default:
		return 1
	}
}

// recordPID appends the child pid to $PROCHUB_TEST_CHILD_PIDFILE so a test can
// count how many children one entry actually spawned.
func recordPID() {
	path := os.Getenv("PROCHUB_TEST_CHILD_PIDFILE")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
	_ = f.Close()
}

func testChildEnv(mode string, extra map[string]string) Environment {
	env := Environment{"PROCHUB_TEST_CHILD": mode}
	for k, v := range extra {
		env[k] = v
	}
	return env
}

func pidFileLines(t *testing.T, path string) []int {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read pid file: %v", err)
	}
	var pids []int
	for _, field := range strings.Fields(string(raw)) {
		if n, err := strconv.Atoi(field); err == nil {
			pids = append(pids, n)
		}
	}
	return pids
}

func waitFor(t *testing.T, timeout time.Duration, desc string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", desc)
}

// TestStartDuringRetryCooldownKeepsSingleChild is the regression test for the
// duplicate-child bug: after a failed attempt the entry sits in its retry
// cooldown, and a Start() during that window used to launch a second run loop
// (and therefore a second child process) for the same entry.
func TestStartDuringRetryCooldownKeepsSingleChild(t *testing.T) {
	dir := t.TempDir()
	flag := filepath.Join(dir, "flag")
	pidFile := filepath.Join(dir, "pids")

	m := NewManager()
	m.Register(Definition{
		ID:      "proc-1",
		Name:    "demo",
		Command: os.Args[0],
		Env: testChildEnv("fail-until-flag", map[string]string{
			"PROCHUB_TEST_CHILD_FLAG":    flag,
			"PROCHUB_TEST_CHILD_PIDFILE": pidFile,
		}),
		RestartPolicy: RestartOnFailure,
	})

	if err := m.Start(context.Background(), "proc-1"); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// First attempt fails; the loop enters its retry cooldown.
	waitFor(t, 5*time.Second, "the first failed attempt to be recorded", func() bool {
		snap, err := m.Get("proc-1")
		return err == nil && snap.Restarts >= 1
	})

	// A second Start during the cooldown must be a no-op, not a second loop.
	if err := m.Start(context.Background(), "proc-1"); err != nil {
		t.Fatalf("second Start: %v", err)
	}

	// Let the pending retry succeed and stay alive, then count the children.
	if err := os.WriteFile(flag, []byte("go"), 0o644); err != nil {
		t.Fatalf("write flag: %v", err)
	}
	time.Sleep(3 * time.Second)

	if pids := pidFileLines(t, pidFile); len(pids) != 1 {
		t.Fatalf("expected exactly 1 child process for one entry, got %d: %v", len(pids), pids)
	}

	_ = m.Stop("proc-1")
}

// TestStopDuringRetryCooldownEndsLoop guards the companion problem: a stop while
// the loop was waiting to retry used to be undone by the loop (which reset its
// manual-stop flag and spawned again).
func TestStopDuringRetryCooldownEndsLoop(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pids")

	m := NewManager()
	m.Register(Definition{
		ID:      "proc-1",
		Name:    "demo",
		Command: os.Args[0],
		Env: testChildEnv("count-fail", map[string]string{
			"PROCHUB_TEST_CHILD_PIDFILE": pidFile,
		}),
		RestartPolicy: RestartOnFailure,
	})

	if err := m.Start(context.Background(), "proc-1"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitFor(t, 5*time.Second, "the first failed attempt", func() bool {
		snap, err := m.Get("proc-1")
		return err == nil && snap.Restarts >= 1
	})

	if err := m.Stop("proc-1"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	before := len(pidFileLines(t, pidFile))

	time.Sleep(3 * time.Second)
	if after := len(pidFileLines(t, pidFile)); after != before {
		t.Fatalf("process respawned after Stop: %d -> %d attempts", before, after)
	}

	snap, err := m.Get("proc-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if snap.Status != StatusStopped {
		t.Fatalf("expected status stopped after Stop, got %s", snap.Status)
	}

	// The entry must be startable again once the previous loop has ended.
	if err := m.Start(context.Background(), "proc-1"); err != nil {
		t.Fatalf("Start after Stop: %v", err)
	}
	waitFor(t, 5*time.Second, "a new attempt after the restart", func() bool {
		return len(pidFileLines(t, pidFile)) > before
	})
	_ = m.Stop("proc-1")
}

// TestStopThenStartLeavesSingleChild covers the RestartProcess sequence
// (Stop immediately followed by Start): the first child must be gone before the
// next one starts, and the entry must never hold two living children.
func TestStopThenStartLeavesSingleChild(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pids")

	m := NewManager()
	m.Register(Definition{
		ID:      "proc-1",
		Name:    "demo",
		Command: os.Args[0],
		Env: testChildEnv("run", map[string]string{
			"PROCHUB_TEST_CHILD_PIDFILE": pidFile,
		}),
		RestartPolicy: RestartAlways,
	})

	if err := m.Start(context.Background(), "proc-1"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitFor(t, 5*time.Second, "the first child", func() bool {
		return len(pidFileLines(t, pidFile)) == 1
	})
	first := pidFileLines(t, pidFile)[0]

	if err := m.Stop("proc-1"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if platform.ProcessAlive(first) {
		t.Fatalf("child %d is still alive after Stop", first)
	}

	if err := m.Start(context.Background(), "proc-1"); err != nil {
		t.Fatalf("Start after Stop: %v", err)
	}
	waitFor(t, 5*time.Second, "the second child", func() bool {
		return len(pidFileLines(t, pidFile)) == 2
	})

	time.Sleep(500 * time.Millisecond)
	pids := pidFileLines(t, pidFile)
	if len(pids) != 2 {
		t.Fatalf("expected 2 total spawns, got %d: %v", len(pids), pids)
	}
	if !platform.ProcessAlive(pids[1]) {
		t.Fatalf("second child %d should be alive", pids[1])
	}

	_ = m.Stop("proc-1")
}
