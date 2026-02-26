package process

import "time"

type Status string

const (
	StatusRunning  Status = "running"
	StatusStopped  Status = "stopped"
	StatusErrored  Status = "errored"
	StatusStarting Status = "starting"
)

type RestartPolicy string

const (
	RestartAlways    RestartPolicy = "always"
	RestartOnFailure RestartPolicy = "on_failure"
	RestartNever     RestartPolicy = "never"
)

type Environment map[string]string

type Definition struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Command       string        `json:"command"`
	Args          []string      `json:"args"`
	WorkingDir    string        `json:"workingDir"`
	Env           Environment   `json:"env"`
	AutoStart     bool          `json:"autoStart"`     // Auto-start on app launch
	AutoRestart   bool          `json:"autoRestart"`   // Deprecated: use RestartPolicy
	RestartPolicy RestartPolicy `json:"restartPolicy"` // Restart policy
	MaxRetries    int           `json:"maxRetries"`    // Max restart attempts
}

type Snapshot struct {
	Definition Definition `json:"definition"`
	PID        int        `json:"pid"`
	Status     Status     `json:"status"`
	Restarts   int        `json:"restarts"`
	LastError  string     `json:"lastError"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	StoppedAt  *time.Time `json:"stoppedAt,omitempty"`
}

// ProcessStats contains resource usage statistics
type ProcessStats struct {
	PID        int     `json:"pid"`
	CPUPercent float64 `json:"cpuPercent"`
	MemoryMB   float64 `json:"memoryMB"`
	Uptime     int64   `json:"uptime"` // seconds
}
