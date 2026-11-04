package updater

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ProcessesRunningError carries the names of the processes blocking an install.
type ProcessesRunningError struct {
	Names []string
}

func (e *ProcessesRunningError) Error() string {
	return fmt.Sprintf("%v: %s", ErrProcessesRunning, strings.Join(e.Names, ", "))
}

func (e *ProcessesRunningError) Is(target error) bool {
	return target == ErrProcessesRunning
}

// Options configures a Manager.
type Options struct {
	Endpoint         string
	Repo             string
	CurrentVersion   string
	Channel          string
	UUID             string
	Platform         Platform
	DataDir          string
	UserAgent        string
	OnEvent          func(Event)
	RunningProcesses func() []string
}

// Manager owns the check/download/install state machine.
type Manager struct {
	opMu sync.Mutex

	mu       sync.Mutex
	opts     Options
	client   *Client
	state    State
	release  *Release
	asset    *Asset
	filePath string
}

// New creates an update manager. It does not touch the network until Check.
func New(opts Options) *Manager {
	if opts.Channel == "" {
		// latest follows GitHub publish time and includes betas. Users still
		// get a version comparison, so an older beta never downgrades them.
		opts.Channel = "latest"
	}
	if opts.Platform.Name == "" || opts.Platform.Arch == "" {
		opts.Platform = CurrentPlatform()
	}
	if opts.UserAgent == "" {
		opts.UserAgent = "ProcHub-Updater"
	}
	return &Manager{
		opts:   opts,
		client: NewClient(opts.Endpoint, opts.Repo, opts.Platform, opts.UserAgent),
		state: State{
			State:          StateIdle,
			CurrentVersion: opts.CurrentVersion,
			Channel:        opts.Channel,
		},
	}
}

// State returns a copy of the current state.
func (m *Manager) State() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// SetChannel switches between stable and beta and clears any fetched release.
func (m *Manager) SetChannel(channel string) {
	if channel != "beta" && channel != "stable" {
		channel = "latest"
	}
	m.mu.Lock()
	m.opts.Channel = channel
	m.release = nil
	m.asset = nil
	m.filePath = ""
	m.state = State{
		State:          StateIdle,
		CurrentVersion: m.opts.CurrentVersion,
		Channel:        channel,
		UpdatedAt:      time.Now().Format(time.RFC3339),
	}
	m.mu.Unlock()
	m.emit()
}

// Check queries the backend (or GitHub) for the newest release.
func (m *Manager) Check(ctx context.Context) (State, error) {
	m.opMu.Lock()
	defer m.opMu.Unlock()

	m.mu.Lock()
	current := m.opts.CurrentVersion
	channel := m.opts.Channel
	uuid := m.opts.UUID
	m.state.State = StateChecking
	m.state.Error = ""
	m.mu.Unlock()
	m.emit()

	release, err := m.client.Check(ctx, current, channel, uuid)
	if err != nil {
		return m.fail(fmt.Errorf("check update: %w", err))
	}

	m.mu.Lock()
	m.release = release
	m.state.LatestVersion = release.Version
	m.state.Release = release
	m.state.UpdatedAt = time.Now().Format(time.RFC3339)
	asset := PickAsset(release, m.opts.Platform)
	if CompareVersions(release.Version, current) <= 0 {
		m.asset = nil
		m.state.State = StateUpToDate
		m.state.Asset = nil
		state := m.state
		m.mu.Unlock()
		m.emit()
		return state, nil
	}
	if asset != nil && asset.SHA256 == "" {
		// Without a hash the download cannot be verified. Do not offer an
		// automatic install; send the user to the download page instead.
		asset = nil
	}
	if asset == nil {
		// A newer version exists but every asset needs a manual install (dmg,
		// deb) or has no verifiable hash. Keep the release so the UI can open
		// the download page.
		if len(release.Assets) > 0 {
			m.asset = nil
			m.state.State = StateAvailable
			m.state.Asset = nil
			state := m.state
			m.mu.Unlock()
			m.emit()
			return state, nil
		}
		m.asset = nil
		m.state.State = StateFailed
		m.state.Error = "no compatible asset for this platform"
		state := m.state
		m.mu.Unlock()
		m.emit()
		return state, fmt.Errorf("no compatible asset for %s/%s", m.opts.Platform.Name, m.opts.Platform.Arch)
	}
	m.asset = asset
	m.state.Asset = asset
	m.state.State = StateAvailable
	state := m.state
	m.mu.Unlock()
	m.emit()
	return state, nil
}

// Download fetches the selected asset and verifies its hash.
func (m *Manager) Download(ctx context.Context) (State, error) {
	m.opMu.Lock()
	defer m.opMu.Unlock()

	m.mu.Lock()
	asset := m.asset
	if m.state.State != StateAvailable && m.state.State != StateFailed {
		m.mu.Unlock()
		return m.state, fmt.Errorf("nothing to download (state %s)", m.state.State)
	}
	if asset == nil {
		m.mu.Unlock()
		return m.state, fmt.Errorf("no asset selected")
	}
	dir := filepath.Join(m.opts.DataDir, "updates")
	if m.opts.DataDir == "" {
		dir = "updates"
	}
	m.state.State = StateDownloading
	m.state.Progress = 0
	m.state.Downloaded = 0
	m.state.Total = asset.Size
	m.state.Error = ""
	m.mu.Unlock()
	m.emit()

	urls := OrderBySpeed(ctx, asset.URLs, m.client.UserAgent)
	ordered := *asset
	ordered.URLs = urls
	path, err := Download(ctx, ordered, dir, m.client.UserAgent, func(done, total int64) {
		m.mu.Lock()
		m.state.Downloaded = done
		m.state.Total = total
		if total > 0 {
			m.state.Progress = int(done * 100 / total)
		}
		state := m.state
		m.mu.Unlock()
		m.emitState(state)
	})
	if err != nil {
		return m.fail(fmt.Errorf("download: %w", err))
	}

	m.mu.Lock()
	m.filePath = path
	m.state.DownloadedPath = path
	m.state.Progress = 100
	m.state.State = StateReady
	state := m.state
	m.mu.Unlock()
	m.emit()
	return state, nil
}

// Install replaces the current installation. It refuses to run while managed
// processes are still active: updating means a full stop, like quitting.
func (m *Manager) Install(ctx context.Context) (InstallResult, State, error) {
	m.opMu.Lock()
	defer m.opMu.Unlock()

	m.mu.Lock()
	if m.state.State != StateReady && m.state.State != StateFailed {
		m.mu.Unlock()
		return InstallResult{}, m.State(), fmt.Errorf("nothing to install (state %s)", m.state.State)
	}
	asset := m.asset
	path := m.filePath
	m.mu.Unlock()

	if asset == nil || path == "" {
		state, _ := m.fail(fmt.Errorf("install: missing downloaded asset"))
		return InstallResult{}, state, fmt.Errorf("missing downloaded asset")
	}

	// Check processes before changing the state: a blocked install must keep
	// the "ready" state so it can be retried once the user stops them.
	if m.opts.RunningProcesses != nil {
		if names := m.opts.RunningProcesses(); len(names) > 0 {
			return InstallResult{}, m.State(), &ProcessesRunningError{Names: names}
		}
	}

	m.mu.Lock()
	m.state.State = StateInstalling
	m.state.Error = ""
	m.mu.Unlock()
	m.emit()

	result, err := Install(ctx, *asset, path, InstallOptions{})
	if err != nil {
		state, _ := m.fail(fmt.Errorf("install: %w", err))
		return InstallResult{}, state, err
	}

	m.mu.Lock()
	if result.RestartRequired {
		m.state.State = StateRestarting
	}
	m.state.UpdatedAt = time.Now().Format(time.RFC3339)
	state := m.state
	m.mu.Unlock()
	m.emit()
	return result, state, nil
}

func (m *Manager) fail(err error) (State, error) {
	m.mu.Lock()
	m.state.State = StateFailed
	m.state.Error = err.Error()
	m.state.UpdatedAt = time.Now().Format(time.RFC3339)
	state := m.state
	m.mu.Unlock()
	m.emit()
	return state, err
}

func (m *Manager) emit() {
	m.mu.Lock()
	state := m.state
	cb := m.opts.OnEvent
	m.mu.Unlock()
	if cb != nil {
		cb(Event{Type: "update:state", State: state})
	}
}

func (m *Manager) emitState(state State) {
	m.mu.Lock()
	cb := m.opts.OnEvent
	m.mu.Unlock()
	if cb != nil {
		cb(Event{Type: "update:state", State: state})
	}
}
