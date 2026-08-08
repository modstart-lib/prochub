package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// appDirName is the application root directory name under the user home.
// It holds ~/.prochub/client.json which stores the data root override.
const appDirName = ".prochub"

// dataDirName is the default data directory name under the app root.
const dataDirName = "data"

// clientConfigName is the file that stores the dataRoot override.
const clientConfigName = "client.json"

// envDataRoot is the environment variable that overrides the data directory.
// The packaged release can inject it (e.g. via LSEnvironment in Info.plist)
// to keep real user data isolated from the default directory used by dev
// tests and seed scripts.
const envDataRoot = "PROCHUB_DATA_ROOT"

// clientConfig mirrors ~/.prochub/client.json.
type clientConfig struct {
	// DataRoot overrides the default data directory (supports ~/ expansion).
	DataRoot string `json:"dataRoot"`
}

// DataDir returns the application data directory and creates it if it does
// not exist yet.
// Priority: PROCHUB_DATA_ROOT env var > client.json dataRoot > ~/.prochub/data.
func DataDir() (string, error) {
	dir := ""
	if env := dataRootFromEnv(); env != "" {
		dir = env
	} else if cfg, err := loadClientConfig(); err == nil && strings.TrimSpace(cfg.DataRoot) != "" {
		dir = cfg.DataRoot
	}
	if dir == "" {
		defaultRoot, err := defaultDataRoot()
		if err != nil {
			return "", err
		}
		dir = defaultRoot
	}
	resolved, err := filepath.Abs(expandHome(dir))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(resolved, 0o755); err != nil {
		return "", err
	}
	return resolved, nil
}

// MustDataDir is like DataDir but never returns an error: it falls back to a
// directory under the OS temp dir when the user home cannot be resolved.
func MustDataDir() string {
	dir, err := DataDir()
	if err != nil {
		return filepath.Join(os.TempDir(), appDirName, dataDirName)
	}
	return dir
}

// dataRootFromEnv resolves the PROCHUB_DATA_ROOT override. It returns an
// empty string when the variable is unset or blank.
func dataRootFromEnv() string {
	return strings.TrimSpace(os.Getenv(envDataRoot))
}

// loadClientConfig reads ~/.prochub/client.json, creating it with the default
// dataRoot when it does not exist. On any read/parse error it falls back to
// the default config so the app always starts.
func loadClientConfig() (clientConfig, error) {
	filePath, err := clientConfigPath()
	if err != nil {
		return clientConfig{}, err
	}
	defaultCfg := clientConfig{}
	if root, err := defaultDataRoot(); err == nil {
		defaultCfg.DataRoot = root
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := writeClientConfig(filePath, defaultCfg); err != nil {
			return clientConfig{}, err
		}
		return defaultCfg, nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return clientConfig{}, err
	}
	var cfg clientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultCfg, nil
	}
	if strings.TrimSpace(cfg.DataRoot) == "" {
		return defaultCfg, nil
	}
	return cfg, nil
}

// clientConfigPath returns the path of ~/.prochub/client.json, creating the
// app root directory if needed.
func clientConfigPath() (string, error) {
	root, err := appRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, clientConfigName), nil
}

// appRootDir returns ~/.prochub, creating it if it does not exist yet.
func appRootDir() (string, error) {
	home, err := homeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, appDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// defaultDataRoot returns the default data directory ~/.prochub/data.
func defaultDataRoot() (string, error) {
	root, err := appRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, dataDirName), nil
}

// writeClientConfig writes the client config file.
func writeClientConfig(filePath string, cfg clientConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0o644)
}

// expandHome expands a leading "~" or "~/" to the user home directory.
func expandHome(value string) string {
	if value == "~" {
		if home, err := homeDir(); err == nil {
			return home
		}
		return value
	}
	if strings.HasPrefix(value, "~/") {
		if home, err := homeDir(); err == nil {
			return filepath.Join(home, value[2:])
		}
	}
	return value
}

// homeDir resolves the user home directory using the standard OS lookup,
// falling back to platform-specific environment variables.
func homeDir() (string, error) {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home, nil
	}
	if home := homeDirFallback(); home != "" {
		return home, nil
	}
	return "", os.ErrNotExist
}
