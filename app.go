package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"prochub/app/config"
	"prochub/app/logging"
	"prochub/app/process"
	"prochub/app/service"
	"prochub/app/store"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	AppName        = "prochub"
	AppDisplayName = "ProcHub"
)

// App struct
type App struct {
	ctx          context.Context
	pm           *process.Manager
	store        *store.Store
	config       config.AppConfig
	logHub       *logging.StreamHub
	loggers      map[string]*ProcessLogger
	autoStartMgr *service.AutoStartManager
}

// ProcessLogger holds the logger for a specific process
type ProcessLogger struct {
	store *logging.RollingStore
	hub   *logging.StreamHub
}

// NewApp creates a new App application struct
func NewApp() *App {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".prochub")

	return &App{
		pm:           process.NewManager(),
		store:        store.NewStore(dataDir),
		logHub:       logging.NewStreamHub(100),
		loggers:      make(map[string]*ProcessLogger),
		autoStartMgr: service.NewAutoStartManager(AppName, AppDisplayName),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load configuration
	cfg, err := a.store.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}
	a.config = cfg

	// Initialize log directory
	if a.config.LogDir == "" {
		a.config.LogDir = "logs"
	}
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".prochub", a.config.LogDir)
	os.MkdirAll(logDir, 0755)

	// Set up log callback for process manager
	a.pm.SetLogCallback(func(processID, stream, line string) {
		logger, ok := a.loggers[processID]
		if !ok {
			return
		}

		entry := logging.Entry{
			Timestamp: time.Now(),
			Stream:    stream,
			Line:      line,
		}

		// Store in memory hub
		logger.hub.Push(entry)

		// Store in rolling file
		logger.store.Append(entry)
	})

	// Register saved processes
	for _, def := range a.config.Processes {
		a.pm.Register(def)

		// Create logger for this process
		processLogDir := filepath.Join(logDir, def.ID)
		a.loggers[def.ID] = &ProcessLogger{
			store: logging.NewRollingStore(processLogDir, a.config.MaxLogLines, a.config.MaxLogFiles),
			hub:   logging.NewStreamHub(100),
		}

		// Auto-start processes if configured
		if def.AutoStart {
			go a.pm.Start(ctx, def.ID)
		}
	}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// AddProcess registers a new process
func (a *App) AddProcess(def process.Definition) error {
	// Generate ID if not provided
	if def.ID == "" {
		def.ID = fmt.Sprintf("proc-%d", len(a.config.Processes)+1)
	}

	// Set default restart policy if not provided
	if def.RestartPolicy == "" {
		def.RestartPolicy = process.RestartOnFailure
	}

	// Register with process manager
	a.pm.Register(def)

	// Create logger for this process
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".prochub", a.config.LogDir, def.ID)
	a.loggers[def.ID] = &ProcessLogger{
		store: logging.NewRollingStore(logDir, a.config.MaxLogLines, a.config.MaxLogFiles),
		hub:   logging.NewStreamHub(100),
	}

	// Add to config and save
	a.config.Processes = append(a.config.Processes, def)
	return a.store.Save(a.config)
}

// RemoveProcess removes a process by ID
func (a *App) RemoveProcess(id string) error {
	// Stop the process first
	a.pm.Stop(id)

	// Remove from config
	newProcesses := make([]process.Definition, 0)
	for _, p := range a.config.Processes {
		if p.ID != id {
			newProcesses = append(newProcesses, p)
		}
	}
	a.config.Processes = newProcesses

	// Remove logger
	delete(a.loggers, id)

	return a.store.Save(a.config)
}

// UpdateProcess updates a process configuration
func (a *App) UpdateProcess(id string, def process.Definition) error {
	// Stop the process first
	a.pm.Stop(id)

	// Update in config
	for i, p := range a.config.Processes {
		if p.ID == id {
			def.ID = id // Preserve the ID
			a.config.Processes[i] = def
			break
		}
	}

	// Re-register with process manager
	a.pm.Register(def)

	// Save config
	return a.store.Save(a.config)
}

// StartProcess starts a process by ID
func (a *App) StartProcess(id string) error {
	return a.pm.Start(a.ctx, id)
}

// StopProcess stops a process by ID
func (a *App) StopProcess(id string) error {
	return a.pm.Stop(id)
}

// RestartProcess restarts a process by ID
func (a *App) RestartProcess(id string) error {
	if err := a.pm.Stop(id); err != nil {
		return err
	}
	return a.pm.Start(a.ctx, id)
}

// ListProcesses returns all processes with their status
func (a *App) ListProcesses() []process.Snapshot {
	return a.pm.List()
}

// GetProcessLogs returns logs for a specific process
func (a *App) GetProcessLogs(id string) []logging.Entry {
	logger, ok := a.loggers[id]
	if !ok {
		return []logging.Entry{}
	}
	return logger.hub.Snapshot()
}

// GetConfig returns the current configuration
func (a *App) GetConfig() config.AppConfig {
	return a.config
}

// UpdateConfig updates the configuration
func (a *App) UpdateConfig(cfg config.AppConfig) error {
	a.config = cfg
	return a.store.Save(a.config)
}

// SelectDirectory opens a directory selection dialog
func (a *App) SelectDirectory() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Working Directory",
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

// SelectFile opens a file selection dialog for selecting executable/command
func (a *App) SelectFile() (string, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Command/Executable",
		Filters: []runtime.FileFilter{
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	return file, nil
}

// GetAutoStartEnabled returns whether auto-start is enabled
func (a *App) GetAutoStartEnabled() (bool, error) {
	return a.autoStartMgr.IsEnabled()
}

// SetAutoStartEnabled enables or disables auto-start
func (a *App) SetAutoStartEnabled(enabled bool) error {
	if enabled {
		return a.autoStartMgr.Enable()
	}
	return a.autoStartMgr.Disable()
}

// GetPlatform returns the current operating system
func (a *App) GetPlatform() string {
	return a.autoStartMgr.GetPlatform()
}

// GetProcess returns a single process by ID
func (a *App) GetProcess(id string) (process.Snapshot, error) {
	return a.pm.Get(id)
}

// AnalyticsEvent represents an analytics event to be sent
type AnalyticsEvent struct {
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// AnalyticsPayload is the request body format for analytics
type AnalyticsPayload struct {
	Data []AnalyticsEvent `json:"data"`
}

const (
	analyticsURL     = "https://open.modstart.com/open_collect"
	appVersion       = "0.1.0"
	analyticsAppName = "ProcHub"
)

// getDeviceUUID returns a persistent UUID for this device
func (a *App) getDeviceUUID() string {
	// Try to load existing UUID from config
	if a.config.DeviceUUID != "" {
		return a.config.DeviceUUID
	}

	// Generate new UUID
	newUUID := uuid.New().String()
	a.config.DeviceUUID = newUUID

	// Save to config
	a.store.Save(a.config)

	return newUUID
}

// getPlatform returns the current platform name
func getPlatform() string {
	switch goruntime.GOOS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	default:
		return goruntime.GOOS
	}
}

// getPlatformArch returns the current platform architecture
func getPlatformArch() string {
	switch goruntime.GOARCH {
	case "amd64":
		return "x64"
	case "arm64":
		return "arm64"
	case "386":
		return "x86"
	default:
		return goruntime.GOARCH
	}
}

// getPlatformVersion returns the OS version
func getPlatformVersion() string {
	switch goruntime.GOOS {
	case "darwin":
		// macOS: use sw_vers command
		out, err := exec.Command("sw_vers", "-productVersion").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	case "windows":
		// Windows: use cmd /c ver
		out, err := exec.Command("cmd", "/c", "ver").Output()
		if err == nil {
			// Parse "Microsoft Windows [Version 10.0.19041.1234]"
			s := string(out)
			if start := strings.Index(s, "[Version "); start != -1 {
				s = s[start+9:]
				if end := strings.Index(s, "]"); end != -1 {
					return strings.TrimSpace(s[:end])
				}
			}
		}
	case "linux":
		// Linux: try /etc/os-release
		data, err := os.ReadFile("/etc/os-release")
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "VERSION_ID=") {
					v := strings.TrimPrefix(line, "VERSION_ID=")
					return strings.Trim(v, "\"")
				}
			}
		}
	}
	return "0"
}

// SendAnalytics sends analytics events to the collection endpoint
func (a *App) SendAnalytics(events []AnalyticsEvent) {
	go func() {
		client := &http.Client{Timeout: 10 * time.Second}

		// Build User-Agent: Open/{AppName}/{Version}/{Platform}/{PlatformArch}/{PlatformVersion}/{UUID}
		userAgent := fmt.Sprintf("Open/%s/%s/%s/%s/%s/%s",
			analyticsAppName,
			appVersion,
			getPlatform(),
			getPlatformArch(),
			getPlatformVersion(),
			a.getDeviceUUID(),
		)

		// Build payload with data array
		payload := AnalyticsPayload{Data: events}
		jsonData, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("[Analytics] Failed to marshal payload: %v\n", err)
			return
		}

		req, err := http.NewRequest("POST", analyticsURL, bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("[Analytics] Failed to create request: %v\n", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[Analytics] Failed to send request: %v\n", err)
			return
		}
		defer resp.Body.Close()

		fmt.Printf("[Analytics] Response status: %d\n", resp.StatusCode)
	}()
}

// SaveLogsToFile opens a save dialog and saves logs to the selected file
func (a *App) SaveLogsToFile(processName string, content string) error {
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Logs",
		DefaultFilename: fmt.Sprintf("%s-logs.txt", processName),
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files", Pattern: "*.txt"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return err
	}
	if filePath == "" {
		return nil // User cancelled
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

// ShowWindow shows the main window (used by system tray)
func (a *App) ShowWindow() {
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
}

// HideWindow hides the main window and Dock icon
func (a *App) HideWindow() {
	runtime.WindowHide(a.ctx)
	// Hide Dock icon on macOS
	HideDockIcon()
}

// QuitApp quits the application
func (a *App) QuitApp() {
	runtime.Quit(a.ctx)
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	// Stop all running processes gracefully
	a.pm.StopAll()
}
