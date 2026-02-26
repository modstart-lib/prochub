package process

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

const (
	// GracefulStopTimeout is the time to wait for graceful process termination
	GracefulStopTimeout = 5 * time.Second
)

var (
	ErrNotFound = errors.New("process not found")
)

// LogCallback is called when process outputs data
type LogCallback func(processID, stream, line string)

type Manager struct {
	mu          sync.RWMutex
	entries     map[string]*entry
	logCallback LogCallback
}

type entry struct {
	definition      Definition
	restarts        int
	status          Status
	cmd             *exec.Cmd
	lastError       string
	manuallyStopped bool // true when stopped by user, false when stopped automatically
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

func (m *Manager) List() []Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshots := make([]Snapshot, 0, len(m.entries))
	for _, item := range m.entries {
		snapshots = append(snapshots, Snapshot{
			Definition: item.definition,
			PID:        pidOf(item.cmd),
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

// Unregister removes a process from the manager
func (m *Manager) Unregister(id string) error {
	m.mu.Lock()
	item, ok := m.entries[id]
	if !ok {
		m.mu.Unlock()
		return ErrNotFound
	}

	// Stop the process if running
	if item.status == StatusRunning || item.status == StatusStarting {
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
		PID:        pidOf(item.cmd),
		Status:     item.status,
		Restarts:   item.restarts,
		LastError:  item.lastError,
	}, nil
}

// StopAll stops all running processes
func (m *Manager) StopAll() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.entries))
	for id, item := range m.entries {
		if item.status == StatusRunning || item.status == StatusStarting {
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

	if item.status == StatusRunning || item.status == StatusStarting {
		m.mu.Unlock()
		return nil
	}

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
	m.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		// Try graceful stop first
		done := make(chan struct{})
		go func() {
			gracefulStop(cmd, done)
		}()

		// Wait for graceful stop or timeout
		select {
		case <-done:
			// Wait a bit for the process to exit gracefully
			timer := time.NewTimer(GracefulStopTimeout)
			defer timer.Stop()

			for {
				select {
				case <-timer.C:
					// Timeout, force kill
					return killProcess(cmd)
				default:
					if !isProcessRunning(cmd.Process.Pid) {
						return nil
					}
					time.Sleep(100 * time.Millisecond)
				}
			}
		case <-time.After(GracefulStopTimeout):
			return killProcess(cmd)
		}
	}
	return nil
}

func (m *Manager) run(ctx context.Context, id string) {
	for {
		m.mu.Lock()
		item, ok := m.entries[id]
		if !ok {
			m.mu.Unlock()
			return
		}

		// Clear previous error on restart
		item.lastError = ""
		item.manuallyStopped = false // Reset manual stop flag when starting

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
		item.status = StatusRunning
		logCb := m.logCallback
		m.mu.Unlock()

		err := cmd.Start()
		if err != nil {
			m.recordError(id, err)
			if !m.shouldRestart(id) {
				return
			}
			m.waitForRetry(id)
			continue
		}

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

		m.mu.Lock()
		// Check if manually stopped - don't auto-restart if user explicitly stopped
		if item.manuallyStopped {
			item.status = StatusStopped
			m.mu.Unlock()
			return
		}
		m.mu.Unlock()

		if !m.shouldRestart(id) {
			return
		}
		m.waitForRetry(id)
	}
}

func (m *Manager) streamOutput(id, stream string, reader io.Reader, callback LogCallback) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
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

func (m *Manager) waitForRetry(id string) {
	m.mu.RLock()
	_, ok := m.entries[id]
	m.mu.RUnlock()
	if !ok {
		return
	}
	// Simple cooldown window to avoid immediate thrash.
	time.Sleep(2 * time.Second)
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

func pidOf(cmd *exec.Cmd) int {
	if cmd == nil || cmd.Process == nil {
		return 0
	}
	return cmd.Process.Pid
}

func envFromMap(extra Environment) []string {
	output := make([]string, 0, len(extra))
	for key, value := range extra {
		output = append(output, key+"="+value)
	}
	return output
}
