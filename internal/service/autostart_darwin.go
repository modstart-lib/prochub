//go:build darwin

package service

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const launchAgentTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>{{if .IsAppBundle}}
        <string>/usr/bin/open</string>
        <string>-a</string>{{end}}
        <string>{{.AppPath}}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
    <key>StandardOutPath</key>
    <string>{{.LogDir}}/{{.AppName}}.out.log</string>
    <key>StandardErrorPath</key>
    <string>{{.LogDir}}/{{.AppName}}.err.log</string>
</dict>
</plist>
`

type launchAgentConfig struct {
	Label       string
	AppName     string
	AppPath     string
	LogDir      string
	IsAppBundle bool
}

// getLaunchAgentPath returns the path to the LaunchAgent plist file
func getLaunchAgentPath(appName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "Library", "LaunchAgents", "com."+appName+".plist"), nil
}

// isAutoStartEnabled checks if auto-start is enabled on macOS
func isAutoStartEnabled(appName, appPath string) (bool, error) {
	plistPath, err := getLaunchAgentPath(appName)
	if err != nil {
		return false, err
	}

	// Check if the plist file exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return false, nil
	}

	// Read and check if it contains our app path
	content, err := os.ReadFile(plistPath)
	if err != nil {
		return false, err
	}

	return strings.Contains(string(content), appPath), nil
}

// enableAutoStart enables auto-start on macOS using LaunchAgent
func enableAutoStart(appName, displayName, appPath string) error {
	plistPath, err := getLaunchAgentPath(appName)
	if err != nil {
		return err
	}

	// Ensure LaunchAgents directory exists
	launchAgentsDir := filepath.Dir(plistPath)
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return err
	}

	// Create log directory
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, "."+appName, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Check if we're running from a .app bundle
	// If the path contains .app/Contents/MacOS/, extract the .app path
	finalAppPath := appPath
	isAppBundle := false
	if strings.Contains(appPath, ".app/Contents/MacOS/") {
		// Extract the .app bundle path
		parts := strings.Split(appPath, ".app/Contents/MacOS/")
		if len(parts) > 0 {
			finalAppPath = parts[0] + ".app"
			isAppBundle = true
		}
	}

	// Generate plist content
	config := launchAgentConfig{
		Label:       "com." + appName,
		AppName:     appName,
		AppPath:     finalAppPath,
		LogDir:      logDir,
		IsAppBundle: isAppBundle,
	}

	tmpl, err := template.New("launchAgent").Parse(launchAgentTemplate)
	if err != nil {
		return err
	}

	file, err := os.Create(plistPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := tmpl.Execute(file, config); err != nil {
		return err
	}

	// Set correct permissions
	return os.Chmod(plistPath, 0644)
}

// disableAutoStart disables auto-start on macOS
func disableAutoStart(appName string) error {
	plistPath, err := getLaunchAgentPath(appName)
	if err != nil {
		return err
	}

	// Remove the plist file if it exists
	if _, err := os.Stat(plistPath); err == nil {
		return os.Remove(plistPath)
	}

	return nil
}
