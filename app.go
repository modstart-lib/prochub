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
	systemLogger *logging.RollingStore
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
		// Initialize system logger first before logging errors
		homeDir, _ := os.UserHomeDir()
		systemLogDir := filepath.Join(homeDir, ".prochub", "system_logs")
		os.MkdirAll(systemLogDir, 0755)
		a.systemLogger = logging.NewRollingStore(systemLogDir, 1000, 10)
		a.LogSystemError("startup", fmt.Sprintf("Failed to load config, using default: %v", err))
	}
	a.config = cfg

	// Initialize log directory
	if a.config.LogDir == "" {
		a.config.LogDir = "logs"
	}
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".prochub", a.config.LogDir)
	os.MkdirAll(logDir, 0755)
	// Initialize system logger
	systemLogDir := filepath.Join(homeDir, ".prochub", "system_logs")
	os.MkdirAll(systemLogDir, 0755)
	a.systemLogger = logging.NewRollingStore(systemLogDir, 1000, 10)
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

	// Log successful startup
	a.LogSystemError("startup", fmt.Sprintf("Application started successfully, version: %s, platform: %s", appConfig.Version, a.autoStartMgr.GetPlatform()))
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
	err := a.store.Save(a.config)
	if err != nil {
		a.LogSystemError("AddProcess", fmt.Sprintf("Failed to save config after adding process %s: %v", def.Name, err))
	}
	return err
}

// RemoveProcess removes a process by ID
func (a *App) RemoveProcess(id string) error {
	// Stop the process first
	err := a.pm.Stop(id)
	if err != nil {
		a.LogSystemError("RemoveProcess", fmt.Sprintf("Failed to stop process %s: %v", id, err))
	}

	// Unregister from process manager
	a.pm.Unregister(id)

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

	err = a.store.Save(a.config)
	if err != nil {
		a.LogSystemError("RemoveProcess", fmt.Sprintf("Failed to save config after removing process %s: %v", id, err))
	}
	return err
}

// UpdateProcess updates a process configuration
func (a *App) UpdateProcess(id string, def process.Definition) error {
	// Stop the process first
	err := a.pm.Stop(id)
	if err != nil {
		a.LogSystemError("UpdateProcess", fmt.Sprintf("Failed to stop process %s: %v", id, err))
	}

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
	err = a.store.Save(a.config)
	if err != nil {
		a.LogSystemError("UpdateProcess", fmt.Sprintf("Failed to save config after updating process %s: %v", id, err))
	}
	return err
}

// StartProcess starts a process by ID
func (a *App) StartProcess(id string) error {
	err := a.pm.Start(a.ctx, id)
	if err != nil {
		a.LogSystemError("StartProcess", fmt.Sprintf("Failed to start process %s: %v", id, err))
	}
	return err
}

// StopProcess stops a process by ID
func (a *App) StopProcess(id string) error {
	err := a.pm.Stop(id)
	if err != nil {
		a.LogSystemError("StopProcess", fmt.Sprintf("Failed to stop process %s: %v", id, err))
	}
	return err
}

// RestartProcess restarts a process by ID
func (a *App) RestartProcess(id string) error {
	if err := a.pm.Stop(id); err != nil {
		a.LogSystemError("RestartProcess", fmt.Sprintf("Failed to stop process %s during restart: %v", id, err))
		return err
	}
	err := a.pm.Start(a.ctx, id)
	if err != nil {
		a.LogSystemError("RestartProcess", fmt.Sprintf("Failed to start process %s during restart: %v", id, err))
	}
	return err
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
	oldLocale := a.config.Locale
	a.config = cfg
	
	// Update tray language if locale changed
	if oldLocale != cfg.Locale {
		UpdateTrayLanguage()
	}
	
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

// GetAppName returns the application display name
func (a *App) GetAppName() string {
	return AppDisplayName
}

// GetAppVersion returns the current app version
func (a *App) GetAppVersion() string {
	return appConfig.Version
}

// GetSystemVersion returns detailed system version information
func (a *App) GetSystemVersion() map[string]string {
	info := make(map[string]string)
	info["os"] = goruntime.GOOS
	info["arch"] = goruntime.GOARCH
	info["platform"] = a.autoStartMgr.GetPlatform()
	
	// Get OS version based on platform
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "darwin":
		cmd = exec.Command("sw_vers", "-productVersion")
	case "linux":
		cmd = exec.Command("lsb_release", "-ds")
		// Fallback to /etc/os-release if lsb_release not available
		if _, err := exec.LookPath("lsb_release"); err != nil {
			cmd = exec.Command("sh", "-c", "cat /etc/os-release | grep PRETTY_NAME | cut -d'=' -f2 | tr -d '\"'")
		}
	case "windows":
		cmd = exec.Command("cmd", "/c", "ver")
	}
	
	if cmd != nil {
		if output, err := cmd.Output(); err == nil {
			info["osVersion"] = strings.TrimSpace(string(output))
		} else {
			info["osVersion"] = "unknown"
		}
	}
	
	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		info["hostname"] = hostname
	}
	
	info["goVersion"] = goruntime.Version()
	info["numCPU"] = fmt.Sprintf("%d", goruntime.NumCPU())
	
	return info
}

// LogSystemError logs system errors to the system log file
func (a *App) LogSystemError(component, message string) {
	if a.systemLogger == nil {
		return
	}
	entry := logging.Entry{
		Timestamp: time.Now(),
		Stream:    component,
		Line:      message,
	}
	a.systemLogger.Append(entry)
}

// GetSystemLogs returns system logs from the last 24 hours
func (a *App) GetSystemLogs() (string, error) {
	var logs strings.Builder
	
	// Collect application system logs
	homeDir, _ := os.UserHomeDir()
	systemLogDir := filepath.Join(homeDir, ".prochub", "system_logs")
	
	logs.WriteString("=== Application System Logs ===\n")
	if entries, err := os.ReadDir(systemLogDir); err == nil {
		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)
		
		totalSize := 0
		maxSize := 500 * 1024 // Limit to 500KB of logs
		
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			
			info, err := entry.Info()
			if err != nil {
				continue
			}
			
			// Only include logs from last 24 hours
			if info.ModTime().After(yesterday) {
				filePath := filepath.Join(systemLogDir, entry.Name())
				content, err := os.ReadFile(filePath)
				if err == nil {
					if totalSize+len(content) > maxSize {
						logs.WriteString(fmt.Sprintf("\n... (remaining logs truncated, limit %dKB reached)\n", maxSize/1024))
						break
					}
					logs.WriteString(fmt.Sprintf("\n--- %s ---\n", entry.Name()))
					logs.Write(content)
					logs.WriteString("\n")
					totalSize += len(content)
				}
			}
		}
		
		if totalSize == 0 {
			logs.WriteString("No system logs found in the last 24 hours\n")
		}
	} else {
		logs.WriteString(fmt.Sprintf("Unable to read system logs directory: %v\n", err))
	}
	
	return logs.String(), nil
}

// GetAppConfig returns application configuration
func (a *App) GetAppConfig() map[string]interface{} {
	return map[string]interface{}{
		"name":            appConfig.Name,
		"title":           appConfig.Title,
		"slogan":          appConfig.Slogan,
		"version":         appConfig.Version,
		"website":         appConfig.Website,
		"websiteGithub":   appConfig.WebsiteGithub,
		"websiteGitee":    appConfig.WebsiteGitee,
		"apiBaseUrl":      appConfig.ApiBaseUrl,
		"analyticsUrl":    appConfig.AnalyticsUrl,
		"versionCheckUrl": appConfig.VersionCheckUrl,
		"feedbackUrl":     appConfig.FeedbackUrl,
		"guideUrl":        appConfig.GuideUrl,
		"helpUrl":         appConfig.HelpUrl,
	}
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

const baseURL = "https://open.modstart.com/open_app/ProcHub"

// AppConfig holds application-wide configuration
var appConfig = struct {
	Name        string
	Title       string
	Slogan      string
	Version     string
	Website     string
	WebsiteGithub string
	WebsiteGitee  string
	ApiBaseUrl    string
	AnalyticsUrl  string
	VersionCheckUrl string
	FeedbackUrl   string
	GuideUrl      string
	HelpUrl       string
}{
	Name:            "ProcHub",
	Title:           "ProcHub",
	Slogan:          "Manage processes easily",
	Version:         "v0.5.1",
	Website:         baseURL,
	WebsiteGithub:   "https://github.com/modstart-lib/prochub",
	WebsiteGitee:    "https://gitee.com/modstart-lib/prochub",
	ApiBaseUrl:      baseURL + "/api",
	AnalyticsUrl:    baseURL + "/app_manager/collect",
	VersionCheckUrl: baseURL + "/app_manager/updater",
	FeedbackUrl:     baseURL + "/feedback_ticket",
	GuideUrl:        baseURL + "/app_manager/guide",
	HelpUrl:         baseURL + "/app_manager/help",
}

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
		return "mac"
	case "windows":
		return "win"
	case "linux":
		return "linux"
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

		// Build User-Agent: AppOpen/{AppName}/{Version} Platform/{PlatformName}/{PlatformArch}/{PlatformVersion}/{UUID}
		userAgent := fmt.Sprintf("AppOpen/%s/%s Platform/%s/%s/%s/%s",
			appConfig.Name,
			appConfig.Version,
			getPlatform(),
			getPlatformArch(),
			getPlatformVersion(),
			a.getDeviceUUID(),
		)

		// Build form data
		formData := map[string]interface{}{
			"uuid":    a.getDeviceUUID(),
			"version": appConfig.Version,
			"data":    events,
			"platform": map[string]string{
				"name":    getPlatform(),
				"arch":    getPlatformArch(),
				"version": getPlatformVersion(),
			},
		}

		jsonData, err := json.Marshal(formData)
		if err != nil {
			fmt.Printf("[Analytics] Failed to marshal payload: %v\n", err)
			return
		}

		req, err := http.NewRequest("POST", appConfig.AnalyticsUrl, bytes.NewBuffer(jsonData))
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

// VersionInfo represents version information from the server
type VersionInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Time    string `json:"time"`
	Url     string `json:"url,omitempty"`
}

// versionCheckResponse is the API response format
type versionCheckResponse struct {
	Code int         `json:"code"`
	Data VersionInfo `json:"data"`
}

// CheckVersion checks for new version from the server
func (a *App) CheckVersion() (VersionInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Build User-Agent: AppOpen/{AppName}/{Version} Platform/{PlatformName}/{PlatformArch}/{PlatformVersion}/{UUID}
	userAgent := fmt.Sprintf("AppOpen/%s/%s Platform/%s/%s/%s/%s",
		appConfig.Name,
		appConfig.Version,
		getPlatform(),
		getPlatformArch(),
		getPlatformVersion(),
		a.getDeviceUUID(),
	)

	// Build form data
	formData := map[string]interface{}{
		"uuid":    a.getDeviceUUID(),
		"version": appConfig.Version,
		"platform": map[string]string{
			"name":    getPlatform(),
			"arch":    getPlatformArch(),
			"version": getPlatformVersion(),
		},
	}

	jsonData, err := json.Marshal(formData)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", appConfig.VersionCheckUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return VersionInfo{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return VersionInfo{}, fmt.Errorf("HTTP error: status %d", resp.StatusCode)
	}

	var result versionCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return VersionInfo{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Code != 0 {
		return VersionInfo{}, fmt.Errorf("API error: code %d", result.Code)
	}

	return result.Data, nil
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	// Log shutdown
	a.LogSystemError("shutdown", "Application is shutting down")
	
	// Stop all running processes gracefully
	a.pm.StopAll()
	
	// Final log
	a.LogSystemError("shutdown", "Application shutdown complete")
}
