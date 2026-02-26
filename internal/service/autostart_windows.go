//go:build windows

package service

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const registryKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// isAutoStartEnabled checks if auto-start is enabled on Windows
func isAutoStartEnabled(appName, appPath string) (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer key.Close()

	val, _, err := key.GetStringValue(appName)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}

	// Check if the registered path matches our app path
	// Windows paths are case-insensitive
	return strings.EqualFold(val, appPath) || strings.EqualFold(val, `"`+appPath+`"`), nil
}

// enableAutoStart enables auto-start on Windows using Registry
func enableAutoStart(appName, displayName, appPath string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, registryKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	// Quote the path to handle spaces
	return key.SetStringValue(appName, `"`+appPath+`"`)
}

// disableAutoStart disables auto-start on Windows
func disableAutoStart(appName string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryKeyPath, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	defer key.Close()

	err = key.DeleteValue(appName)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}

// getStartupFolderPath returns the path to the Windows Startup folder
// This is an alternative method for auto-start
func getStartupFolderPath() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		appData = filepath.Join(homeDir, "AppData", "Roaming")
	}
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup"), nil
}
