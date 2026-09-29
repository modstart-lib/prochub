package control

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// AuthFileName is the auth file name stored in the application data directory.
const AuthFileName = "auth.json"

// AuthInfo tells a CLI client how to reach the running application's control
// server. The GUI writes it on startup and removes it on shutdown.
type AuthInfo struct {
	Port      int       `json:"port"`
	Token     string    `json:"token"`
	PID       int       `json:"pid"`
	Version   string    `json:"version"`
	StartedAt time.Time `json:"startedAt"`
}

// AuthPath returns the auth file path inside dataDir.
func AuthPath(dataDir string) string {
	return filepath.Join(dataDir, AuthFileName)
}

// NewToken generates a random 32-byte hex token used to authenticate CLI calls.
func NewToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// WriteAuth writes the auth file with owner-only permissions.
func WriteAuth(dataDir string, info AuthInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(AuthPath(dataDir), data, 0o600)
}

// ReadAuth reads the auth file. A missing file returns an os.ErrNotExist error.
func ReadAuth(dataDir string) (AuthInfo, error) {
	data, err := os.ReadFile(AuthPath(dataDir))
	if err != nil {
		return AuthInfo{}, err
	}
	var info AuthInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return AuthInfo{}, err
	}
	return info, nil
}

// RemoveAuth deletes the auth file, ignoring a missing file.
func RemoveAuth(dataDir string) {
	_ = os.Remove(AuthPath(dataDir))
}
