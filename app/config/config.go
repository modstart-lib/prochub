package config

import "prochub/app/process"

type AppConfig struct {
	Locale        string               `json:"locale"`
	AutoStart     bool                 `json:"autoStart"`
	LogDir        string               `json:"logDir"`
	MaxLogLines   int                  `json:"maxLogLines"`
	MaxLogFiles   int                  `json:"maxLogFiles"`
	MaxRestart    int                  `json:"maxRestart"`
	RestartPolicy string               `json:"restartPolicy"`
	Processes     []process.Definition `json:"processes"`
}

func DefaultConfig() AppConfig {
	return AppConfig{
		Locale:        "zh",
		AutoStart:     false,
		LogDir:        "logs",
		MaxLogLines:   1000,
		MaxLogFiles:   5,
		MaxRestart:    5,
		RestartPolicy: "on_failure",
	}
}
