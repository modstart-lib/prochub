package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"prochub/internal/config"
	"prochub/internal/control"
	"prochub/internal/logging"
	"prochub/internal/process"
)

// controlBridge exposes the application to the local control server. It is a
// separate type so its methods are not bound to the frontend by Wails.
type controlBridge struct {
	app *App
}

func (b *controlBridge) Status() control.StatusInfo {
	items := b.app.pm.List()
	running := 0
	for _, item := range items {
		if item.Status == process.StatusRunning || item.Status == process.StatusStarting {
			running++
		}
	}
	return control.StatusInfo{
		Version:      appConfig.Version,
		Platform:     b.app.autoStartMgr.GetPlatform(),
		PID:          os.Getpid(),
		ProcessCount: len(items),
		RunningCount: running,
		DataDir:      b.app.dataDir,
	}
}

func (b *controlBridge) ListProcesses() []process.Snapshot {
	return b.app.pm.List()
}

func (b *controlBridge) StartProcess(id string) error {
	return b.app.StartProcess(id)
}

func (b *controlBridge) StopProcess(id string) error {
	return b.app.StopProcess(id)
}

func (b *controlBridge) Logs(id string, tail int) ([]string, error) {
	if _, err := b.app.pm.Get(id); err != nil {
		return nil, err
	}
	logDir := b.app.config.LogDir
	if logDir == "" {
		logDir = "logs"
	}
	return logging.TailFiles(filepath.Join(b.app.dataDir, logDir, id), tail)
}

func (b *controlBridge) Config() control.ConfigSummary {
	return control.ConfigSummary{
		Theme:        b.app.config.Theme,
		Locale:       b.app.config.Locale,
		AutoStart:    b.app.config.AutoStart,
		AutoStartWSL: b.app.config.AutoStartWSL,
		LogDir:       b.app.config.LogDir,
	}
}

func (b *controlBridge) UpdateConfig(patch control.ConfigPatch) error {
	// Patch through a dedicated helper that holds the config lock across the
	// whole read-modify-write, so two concurrent CLI calls patching different
	// fields cannot overwrite each other.
	return b.app.patchConfig(func(cfg *config.AppConfig) {
		if patch.Theme != nil {
			cfg.Theme = *patch.Theme
		}
		if patch.Locale != nil {
			cfg.Locale = *patch.Locale
		}
		if patch.AutoStart != nil {
			cfg.AutoStart = *patch.AutoStart
		}
		if patch.AutoStartWSL != nil {
			cfg.AutoStartWSL = *patch.AutoStartWSL
		}
	}, originCLI)
}

func (b *controlBridge) WSLStatus() control.WSLStatusInfo {
	status := b.app.GetWSLStatus()
	return control.WSLStatusInfo{
		Supported: status.Supported,
		Available: status.Available,
		Running:   status.Running,
		Starting:  status.Starting,
		Version:   status.Version,
		Distro:    status.DefaultDistro,
	}
}

func (b *controlBridge) StartWSL() error {
	return b.app.StartWSL()
}

func (b *controlBridge) StopWSL() error {
	return b.app.StopWSL()
}

func (b *controlBridge) RestartWSL() error {
	return b.app.RestartWSL()
}

// startControlServer starts the local control server and publishes its port and
// token to the data directory so the CLI can reach the running app.
func (a *App) startControlServer() {
	token, err := control.NewToken()
	if err != nil {
		a.LogSystemError("control", fmt.Sprintf("Failed to generate control token: %v", err))
		return
	}

	srv := control.NewServer(&controlBridge{app: a}, token)
	port, err := srv.Start()
	if err != nil {
		a.LogSystemError("control", fmt.Sprintf("Failed to start control server: %v", err))
		return
	}
	a.controlServer = srv

	if err := control.WriteAuth(a.dataDir, control.AuthInfo{
		Port:      port,
		Token:     token,
		PID:       os.Getpid(),
		Version:   appConfig.Version,
		StartedAt: time.Now(),
	}); err != nil {
		a.LogSystemError("control", fmt.Sprintf("Failed to write auth file: %v", err))
		return
	}
	a.LogSystemError("control", fmt.Sprintf("Control server listening on 127.0.0.1:%d", port))
}

// stopControlServer shuts the control server down and removes the auth file.
func (a *App) stopControlServer() {
	if a.controlServer != nil {
		a.controlServer.Stop()
		a.controlServer = nil
	}
	control.RemoveAuth(a.dataDir)
}
