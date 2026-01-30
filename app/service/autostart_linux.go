//go:build linux

package service

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const desktopEntryTemplate = `[Desktop Entry]
Type=Application
Name={{.DisplayName}}
Exec={{.AppPath}}
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Comment=Start {{.DisplayName}} on login
`

type desktopEntryConfig struct {
	AppName     string
	DisplayName string
	AppPath     string
}

// getAutoStartPath returns the path to the autostart desktop entry
func getAutoStartPath(appName string) (string, error) {
	// Try XDG_CONFIG_HOME first, then fall back to ~/.config
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "autostart", appName+".desktop"), nil
}

// isAutoStartEnabled checks if auto-start is enabled on Linux
func isAutoStartEnabled(appName, appPath string) (bool, error) {
	desktopPath, err := getAutoStartPath(appName)
	if err != nil {
		return false, err
	}

	// Check if the desktop entry file exists
	if _, err := os.Stat(desktopPath); os.IsNotExist(err) {
		return false, nil
	}

	// Read and check if it contains our app path
	content, err := os.ReadFile(desktopPath)
	if err != nil {
		return false, err
	}

	return strings.Contains(string(content), appPath), nil
}

// enableAutoStart enables auto-start on Linux using XDG autostart
func enableAutoStart(appName, displayName, appPath string) error {
	desktopPath, err := getAutoStartPath(appName)
	if err != nil {
		return err
	}

	// Ensure autostart directory exists
	autoStartDir := filepath.Dir(desktopPath)
	if err := os.MkdirAll(autoStartDir, 0755); err != nil {
		return err
	}

	// Generate desktop entry content
	config := desktopEntryConfig{
		AppName:     appName,
		DisplayName: displayName,
		AppPath:     appPath,
	}

	tmpl, err := template.New("desktopEntry").Parse(desktopEntryTemplate)
	if err != nil {
		return err
	}

	file, err := os.Create(desktopPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := tmpl.Execute(file, config); err != nil {
		return err
	}

	// Set correct permissions (executable for desktop entries)
	return os.Chmod(desktopPath, 0755)
}

// disableAutoStart disables auto-start on Linux
func disableAutoStart(appName string) error {
	desktopPath, err := getAutoStartPath(appName)
	if err != nil {
		return err
	}

	// Remove the desktop entry file if it exists
	if _, err := os.Stat(desktopPath); err == nil {
		return os.Remove(desktopPath)
	}

	return nil
}
