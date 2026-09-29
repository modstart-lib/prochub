package main

import (
	"fmt"

	"prochub/internal/config"
	"prochub/internal/process"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ConfigEvent is the payload broadcast to the frontend whenever the persisted
// configuration changes. It carries the full configuration so the UI has a
// single, authoritative source of truth regardless of who made the change
// (GUI, CLI, or tray).
type ConfigEvent struct {
	Config config.AppConfig `json:"config"`
	// Origin tells the UI where the change came from so it can avoid redundant
	// work (e.g. "gui", "cli", "tray").
	Origin string `json:"origin"`
}

// Config origins used in ConfigEvent.Origin.
const (
	originGUI  = "gui"
	originCLI  = "cli"
	originBoot = "boot"
)

// applyConfig is the single write path for application configuration. It
// persists the configuration, applies OS-level side effects (auto-start),
// refreshes the tray language and broadcasts the full configuration to the
// frontend over the "config:changed" event.
//
// The origin argument is only used for the broadcast payload; it does not
// change the behaviour of the write.
func (a *App) applyConfig(cfg config.AppConfig, origin string) error {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	old := a.config

	if cfg.AutoStart != old.AutoStart {
		if cfg.AutoStart {
			if err := a.autoStartMgr.Enable(); err != nil {
				return err
			}
		} else if err := a.autoStartMgr.Disable(); err != nil {
			return err
		}
	}

	normalizeConfig(&cfg)

	a.config = cfg
	if err := a.store.Save(a.config); err != nil {
		a.LogSystemError("applyConfig", fmt.Sprintf("Failed to save config: %v", err))
		return err
	}

	// Tray language depends on the locale and is not driven by the frontend.
	if old.Locale != cfg.Locale {
		UpdateTrayLanguage()
	}

	if !configEqual(old, cfg) {
		if origin == "" {
			origin = originGUI
		}
		a.emit("config:changed", ConfigEvent{Config: cfg, Origin: origin})
	}
	return nil
}

// normalizeConfig clamps the fields that have a restricted value set so an
// invalid value never reaches persistence.
func normalizeConfig(cfg *config.AppConfig) {
	if cfg.Theme != "dark" {
		cfg.Theme = "light"
	}
	if cfg.Locale != "en" {
		cfg.Locale = "zh"
	}
}

// configEqual reports whether two configurations are identical. It is used to
// skip broadcasting when a write did not actually change anything.
func configEqual(a, b config.AppConfig) bool {
	if a.Locale != b.Locale ||
		a.Theme != b.Theme ||
		a.AutoStart != b.AutoStart ||
		a.AutoStartWSL != b.AutoStartWSL ||
		a.LogDir != b.LogDir ||
		a.MaxLogLines != b.MaxLogLines ||
		a.MaxLogFiles != b.MaxLogFiles ||
		a.MaxRestart != b.MaxRestart ||
		a.RestartPolicy != b.RestartPolicy {
		return false
	}
	if len(a.Processes) != len(b.Processes) {
		return false
	}
	for i := range a.Processes {
		if !definitionEqual(a.Processes[i], b.Processes[i]) {
			return false
		}
	}
	return true
}

func definitionEqual(a, b process.Definition) bool {
	if a.ID != b.ID ||
		a.Name != b.Name ||
		a.Command != b.Command ||
		a.WorkingDir != b.WorkingDir ||
		a.AutoStart != b.AutoStart ||
		a.AutoRestart != b.AutoRestart ||
		a.RestartPolicy != b.RestartPolicy ||
		a.MaxRetries != b.MaxRetries {
		return false
	}
	if len(a.Args) != len(b.Args) {
		return false
	}
	for i := range a.Args {
		if a.Args[i] != b.Args[i] {
			return false
		}
	}
	if len(a.Env) != len(b.Env) {
		return false
	}
	for k, v := range a.Env {
		if b.Env[k] != v {
			return false
		}
	}
	return true
}

// patchConfig applies a series of mutations to the current configuration while
// holding the config lock across the whole read-modify-write, then persists the
// result. Callers use it instead of reading a.config, mutating a copy and
// calling applyConfig, which would race with concurrent writers.
func (a *App) patchConfig(mutate func(cfg *config.AppConfig), origin string) error {
	a.configMu.Lock()
	cfg := a.config
	a.configMu.Unlock()

	mutate(&cfg)
	return a.applyConfig(cfg, origin)
}

// SetTheme persists the UI theme (light/dark) and broadcasts the change.
func (a *App) SetTheme(theme string) error {
	return a.patchConfig(func(cfg *config.AppConfig) { cfg.Theme = theme }, originGUI)
}

// SetLocale persists the UI locale (zh/en), refreshing the tray language.
func (a *App) SetLocale(locale string) error {
	return a.patchConfig(func(cfg *config.AppConfig) { cfg.Locale = locale }, originGUI)
}

// emit sends a Wails event to the frontend when the app context is ready.
func (a *App) emit(name string, data interface{}) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, data)
}
