package updater

// Update states exposed to the frontend.
const (
	StateIdle        = "idle"
	StateChecking    = "checking"
	StateUpToDate    = "up-to-date"
	StateAvailable   = "available"
	StateDownloading = "downloading"
	StateReady       = "ready"
	StateInstalling  = "installing"
	StateRestarting  = "restarting"
	StateFailed      = "failed"
)

// Platform describes the requesting device, matching the backend contract.
type Platform struct {
	Name    string `json:"name"`
	Arch    string `json:"arch"`
	Version string `json:"version"`
}

// Asset is one downloadable release file.
type Asset struct {
	OS     string   `json:"os"`
	Arch   string   `json:"arch"`
	Kind   string   `json:"kind"`
	Name   string   `json:"name"`
	Size   int64    `json:"size"`
	SHA256 string   `json:"sha256"`
	URLs   []string `json:"urls"`
}

// Release is the newest published version reported by the backend.
type Release struct {
	Name    string  `json:"name"`
	Version string  `json:"version"`
	Time    string  `json:"time"`
	Summary string  `json:"summary"`
	Notes   string  `json:"notes"`
	URL     string  `json:"url"`
	Channel string  `json:"channel"`
	Source  string  `json:"source,omitempty"`
	Assets  []Asset `json:"assets"`
}

// State is the snapshot returned to the frontend.
type State struct {
	State          string   `json:"state"`
	CurrentVersion string   `json:"currentVersion"`
	LatestVersion  string   `json:"latestVersion"`
	Channel        string   `json:"channel"`
	Release        *Release `json:"release,omitempty"`
	Asset          *Asset   `json:"asset,omitempty"`
	Progress       int      `json:"progress"`
	Downloaded     int64    `json:"downloaded"`
	Total          int64    `json:"total"`
	DownloadedPath string   `json:"downloadedPath,omitempty"`
	Error          string   `json:"error,omitempty"`
	UpdatedAt      string   `json:"updatedAt"`
}

// Event is pushed to the UI whenever the state changes.
type Event struct {
	Type  string `json:"type"`
	State State  `json:"state"`
}

// InstallResult reports what happened after an install step.
type InstallResult struct {
	Message         string `json:"message"`
	RestartRequired bool   `json:"restartRequired"`
}
