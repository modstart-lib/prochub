package process

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"prochub/internal/platform"
)

const (
	// GracefulStopTimeout is the time to wait for graceful process termination
	GracefulStopTimeout = 5 * time.Second
	// retryCooldown is the pause between an automatic restart and the next
	// spawn attempt.
	retryCooldown = 2 * time.Second
	// retryPollInterval is how often the cooldown re-checks for a stop request,
	// so a stop during the cooldown takes effect promptly.
	retryPollInterval = 50 * time.Millisecond
	// loopExitTimeout bounds how long Stop waits for a run loop to end.
	loopExitTimeout = 5 * time.Second
)

var (
	ErrNotFound = errors.New("process not found")
)

// LogCallback is called when process outputs data
type LogCallback func(processID, stream, line string)

// LifecycleAction identifies a process lifecycle transition.
type LifecycleAction string

const (
	// ActionSpawn is emitted whenever a child process is started (successfully
	// or not).
	ActionSpawn LifecycleAction = "spawn"
	// ActionExit is emitted when a child process has exited.
	ActionExit LifecycleAction = "exit"
	// ActionRestart is emitted when the run loop schedules an automatic retry.
	ActionRestart LifecycleAction = "restart"
	// ActionStop is emitted when a stop is requested for an entry.
	ActionStop LifecycleAction = "stop"
)

// LifecycleEvent records one process lifecycle transition for auditing. Without
// it "one entry ended up with two children" can only be reconstructed from the
// OS process list, which loses the trigger and the owning entry.
type LifecycleEvent struct {
	Action  LifecycleAction
	ID      string
	PID     int
	Command string
	Reason  string
}

// LifecycleCallback receives lifecycle events for audit logging.
type LifecycleCallback func(LifecycleEvent)

type Manager struct {
	mu                sync.RWMutex
	entries           map[string]*entry
	logCallback       LogCallback
	lifecycleCallback LifecycleCallback
}

type entry struct {
	definition Definition
	restarts   int
	status     Status
	cmd        *exec.Cmd
	// pid is the process id of the current child, published only after the
	// child started successfully and cleared when the loop ends. It is kept
	// separate from cmd because cmd.Process is written by exec.Cmd.Start() and
	// reading it from another goroutine (List/Get/Stop) would race with Start.
	pid             int
	lastError       string
	manuallyStopped bool // true when stopped by user, false when stopped automatically
	// loopActive is true while a run loop is alive for this entry. It is the
	// authoritative "is this entry started" flag: status is not usable for that
	// because it also reflects non-running states (retry cooldown, errored)
	// during which the loop is still alive and will spawn again.
	loopActive bool
}

func NewManager() *Manager {
	return &Manager{
		entries: make(map[string]*entry),
	}
}

// SetLogCallback sets the callback function for process logs
func (m *Manager) SetLogCallback(cb LogCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCallback = cb
}

// SetLifecycleCallback sets the callback function for lifecycle audit events.
func (m *Manager) SetLifecycleCallback(cb LifecycleCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lifecycleCallback = cb
}

func (m *Manager) emitLifecycle(ev LifecycleEvent) {
	m.mu.RLock()
	cb := m.lifecycleCallback
	m.mu.RUnlock()
	if cb != nil {
		cb(ev)
	}
}

func (m *Manager) List() []Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshots := make([]Snapshot, 0, len(m.entries))
	for _, item := range m.entries {
		snapshots = append(snapshots, Snapshot{
			Definition: item.definition,
			PID:        item.pid,
			Status:     item.status,
			Restarts:   item.restarts,
			LastError:  item.lastError,
		})
	}
	return snapshots
}

func (m *Manager) Register(def Definition) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries[def.ID] = &entry{
		definition: def,
		status:     StatusStopped,
	}
}

// SetAutoStart updates the auto-start flag of a process without disturbing a
// running process (it does not stop or restart the process).
func (m *Manager) SetAutoStart(id string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.entries[id]
	if !ok {
		return ErrNotFound
	}
	item.definition.AutoStart = enabled
	return nil
}

// Unregister removes a process from the manager
func (m *Manager) Unregister(id string) error {
	m.mu.Lock()
	item, ok := m.entries[id]
	if !ok {
		m.mu.Unlock()
		return ErrNotFound
	}

	// Stop the process if a run loop is still alive.
	if item.loopActive {
		m.mu.Unlock()
		_ = m.Stop(id)
		m.mu.Lock()
	}

	delete(m.entries, id)
	m.mu.Unlock()
	return nil
}

// Get returns a snapshot for a specific process
func (m *Manager) Get(id string) (Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.entries[id]
	if !ok {
		return Snapshot{}, ErrNotFound
	}

	return Snapshot{
		Definition: item.definition,
		PID:        item.pid,
		Status:     item.status,
		Restarts:   item.restarts,
		LastError:  item.lastError,
	}, nil
}

// RunningNames returns the names of all processes that are currently running
// or starting. It is used to block quitting while processes are still active.
func (m *Manager) RunningNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.entries))
	for _, item := range m.entries {
		if item.status == StatusRunning || item.status == StatusStarting {
			name := item.definition.Name
			if name == "" {
				name = item.definition.ID
			}
			names = append(names, name)
		}
	}
	return names
}

// StopAll stops all processes
func (m *Manager) StopAll() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.entries))
	for id, item := range m.entries {
		if item.loopActive {
			ids = append(ids, id)
		}
	}
	m.mu.RUnlock()

	for _, id := range ids {
		_ = m.Stop(id)
	}
}

func (m *Manager) Start(ctx context.Context, id string) error {
	m.mu.Lock()
	item, ok := m.entries[id]
	if !ok {
		m.mu.Unlock()
		return ErrNotFound
	}

	// The run loop is the only thing that spawns children, so the entry is
	// already started whenever its loop is alive - regardless of the status
	// field, which also holds non-running values (retry cooldown, errored)
	// while the loop is still alive and will spawn again. Guarding on status
	// alone let a second Start launch a second loop (and a second child).
	if item.loopActive {
		m.mu.Unlock()
		return nil
	}

	item.loopActive = true
	item.manuallyStopped = false // Reset manual stop flag when starting
	item.status = StatusStarting
	m.mu.Unlock()

	go m.run(ctx, id)
	return nil
}

func (m *Manager) Stop(id string) error {
	m.mu.Lock()
	item, ok := m.entries[id]
	if !ok {
		m.mu.Unlock()
		return ErrNotFound
	}

	item.status = StatusStopped
	item.manuallyStopped = true // Mark as manually stopped to prevent auto-restart
	cmd := item.cmd
	pid := item.pid
	def := item.definition
	m.mu.Unlock()

	m.emitLifecycle(LifecycleEvent{
		Action:  ActionStop,
		ID:      id,
		PID:     pid,
		Command: commandLine(def),
		Reason:  "manual",
	})

	// pid > 0 means the child had already started, so cmd.Process is published
	// and safe to read here. When it is still 0 the loop has not finished
	// starting the child yet; the loop itself kills it (see run).
	if cmd != nil && pid > 0 {
		terminate(cmd)
	}

	// Wait for the run loop to end. Without this a stop during the retry
	// cooldown would be undone by the loop, and a following Start (as done by
	// RestartProcess) could race with the loop it is replacing.
	m.waitLoopExit(id)
	return nil
}

// terminate stops a running command: it signals the process tree gracefully and
// force-kills it once the graceful timeout elapses. It returns after the command
// is gone (or after the timeout, if the process cannot be observed as stopped).
func terminate(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	done := make(chan struct{})
	go func() {
		_ = gracefulStop(cmd, done)
	}()

	select {
	case <-done:
	case <-time.After(GracefulStopTimeout):
		_ = killProcess(cmd)
		return
	}

	// The graceful signal was sent; give the process a moment to exit before
	// force-killing it.
	deadline := time.Now().Add(GracefulStopTimeout)
	for time.Now().Before(deadline) {
		if !isProcessRunning(cmd.Process.Pid) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = killProcess(cmd)
}

// waitLoopExit blocks until the entry's run loop has ended, or until the timeout
// elapses. Callers use it after asking a process to stop so the loop cannot
// interfere with a following start.
func (m *Manager) waitLoopExit(id string) {
	deadline := time.Now().Add(loopExitTimeout)
	for {
		m.mu.RLock()
		item, ok := m.entries[id]
		active := ok && item.loopActive
		m.mu.RUnlock()
		if !active || time.Now().After(deadline) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// finishLoop clears the run-loop bookkeeping of an entry. It runs exactly once
// per loop, on every return path, so a concurrent Start sees the entry as
// stopped only after the previous loop has really ended.
func (m *Manager) finishLoop(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.entries[id]
	if !ok {
		return
	}
	item.loopActive = false
	item.cmd = nil
	item.pid = 0
}

func (m *Manager) setStatus(id string, status Status) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if item, ok := m.entries[id]; ok {
		item.status = status
	}
}

func (m *Manager) run(ctx context.Context, id string) {
	// The loop guard is cleared once, on every return path, so Start can rely on
	// it to know whether a loop is still alive.
	defer m.finishLoop(id)

	for {
		m.mu.Lock()
		item, ok := m.entries[id]
		if !ok {
			m.mu.Unlock()
			return
		}

		// A stop requested while the loop was between attempts must end the loop
		// instead of spawning the process again.
		if item.manuallyStopped {
			item.status = StatusStopped
			m.mu.Unlock()
			return
		}

		// Clear previous error on restart
		item.lastError = ""
		reason := "start"
		if item.restarts > 0 {
			reason = "retry"
		}
		policy := item.definition.RestartPolicy
		command := commandLine(item.definition)

		cmd := exec.CommandContext(ctx, item.definition.Command, item.definition.Args...)
		cmd.Dir = item.definition.WorkingDir
		cmd.Env = append(cmd.Env, os.Environ()...)
		cmd.Env = append(cmd.Env, envFromMap(item.definition.Env)...)

		// Set up platform-specific process group for proper child process handling
		setupProcessGroup(cmd)

		// Capture stdout and stderr
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		item.cmd = cmd
		item.pid = 0
		item.status = StatusStarting
		logCb := m.logCallback
		m.mu.Unlock()

		err := cmd.Start()
		if err != nil {
			m.recordError(id, err)
			m.emitLifecycle(LifecycleEvent{
				Action: ActionSpawn, ID: id, Command: command, Reason: "failed: " + err.Error(),
			})
			if !m.shouldRestart(id) {
				return
			}
			if !m.waitForRetry(id) {
				return
			}
			continue
		}

		// Publish the child pid under the lock, which also makes cmd.Process safe
		// to read from Stop. A stop (or removal) that arrived while the child was
		// starting kills it here instead, and the wait path below reaps it, so a
		// stop can never be undone by a child it did not get to see.
		childPID := cmd.Process.Pid
		m.mu.Lock()
		current, ok := m.entries[id]
		stopped := !ok || current.manuallyStopped
		if !stopped {
			current.pid = childPID
			current.status = StatusRunning
		}
		m.mu.Unlock()

		if stopped {
			_ = killProcess(cmd)
		}

		m.emitLifecycle(LifecycleEvent{
			Action: ActionSpawn, ID: id, PID: childPID, Command: command, Reason: reason,
		})

		// Stream stdout
		if stdout != nil && logCb != nil {
			go m.streamOutput(id, "stdout", stdout, logCb)
		}

		// Stream stderr
		if stderr != nil && logCb != nil {
			go m.streamOutput(id, "stderr", stderr, logCb)
		}

		err = cmd.Wait()
		if err != nil {
			m.recordError(id, err)
		}

		// Re-read the manual-stop flag after the wait: a stop (or removal) that
		// arrived while the process was exiting must end the loop.
		m.mu.Lock()
		current, ok = m.entries[id]
		stopped = !ok || current.manuallyStopped
		m.mu.Unlock()

		exitReason := "exit 0"
		if err != nil {
			exitReason = err.Error()
		}
		m.emitLifecycle(LifecycleEvent{
			Action: ActionExit, ID: id, PID: childPID, Command: command, Reason: exitReason,
		})

		if stopped {
			m.setStatus(id, StatusStopped)
			return
		}

		if !m.shouldRestart(id) {
			return
		}
		if policy == "" {
			policy = RestartOnFailure
		}
		m.emitLifecycle(LifecycleEvent{
			Action: ActionRestart, ID: id, PID: childPID, Command: command,
			Reason: "policy=" + string(policy),
		})
		if !m.waitForRetry(id) {
			return
		}
	}
}

func (m *Manager) streamOutput(id, stream string, reader io.Reader, callback LogCallback) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		// Console programs on Windows emit legacy-encoded bytes when their
		// output is piped, so decode to UTF-8 before the line is stored.
		line := platform.DecodeOutput(scanner.Bytes())
		if callback != nil {
			callback(id, stream, line)
		}
	}
}

func (m *Manager) shouldRestart(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.entries[id]
	if !ok {
		return false
	}

	policy := item.definition.RestartPolicy
	if policy == "" {
		policy = RestartOnFailure
	}

	if policy == RestartNever {
		item.status = StatusStopped
		return false
	}

	if policy == RestartOnFailure && item.lastError == "" {
		item.status = StatusStopped
		return false
	}

	item.restarts++
	if item.definition.MaxRetries > 0 && item.restarts > item.definition.MaxRetries {
		item.status = StatusErrored
		return false
	}

	return true
}

// waitForRetry waits out the restart cooldown. It returns false when the entry
// disappeared or a stop was requested while waiting, so the caller ends the loop
// instead of spawning again.
func (m *Manager) waitForRetry(id string) bool {
	m.mu.Lock()
	item, ok := m.entries[id]
	if !ok {
		m.mu.Unlock()
		return false
	}
	if item.manuallyStopped {
		m.mu.Unlock()
		return false
	}
	// The entry is waiting to retry, not failed: report it as starting so the UI
	// offers "stop" instead of "start" while a child is imminent.
	item.status = StatusStarting
	m.mu.Unlock()

	deadline := time.Now().Add(retryCooldown)
	for {
		m.mu.RLock()
		current, ok := m.entries[id]
		stopped := !ok || current.manuallyStopped
		m.mu.RUnlock()
		if stopped {
			return false
		}
		if !time.Now().Before(deadline) {
			return true
		}
		time.Sleep(retryPollInterval)
	}
}

// commandLine renders a definition's command with its arguments for audit logs.
func commandLine(def Definition) string {
	if len(def.Args) == 0 {
		return def.Command
	}
	return def.Command + " " + strings.Join(def.Args, " ")
}

func (m *Manager) recordError(id string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.entries[id]
	if !ok {
		return
	}
	item.lastError = err.Error()
	item.status = StatusErrored
}

func envFromMap(extra Environment) []string {
	output := make([]string, 0, len(extra))
	for key, value := range extra {
		output = append(output, key+"="+value)
	}
	return output
}
