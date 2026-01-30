# Prochub

## About

ProcHub is a cross-platform desktop application for process management, built with Wails, Go, and Vue 3 + TypeScript. It provides an intuitive interface to manage, monitor, and control background processes across Windows, macOS, and Linux.

![Build Status](https://github.com/yourusername/prochub/workflows/Build%20and%20Release/badge.svg)

You can configure the project by editing `wails.json`. More information about the project settings can be found
here: https://wails.io/docs/reference/project-config

## Features

### Process Management
- **Add/Remove Processes**: Easily add new processes with customizable commands, arguments, and environment variables
- **Start/Stop/Restart**: Full control over process lifecycle with graceful shutdown support
- **Auto-start**: Configure processes to start automatically when the application launches
- **Restart Policies**: Support for `always`, `on_failure`, and `never` restart policies
- **Process Monitoring**: Real-time status monitoring with PID, restart count, and error tracking

### Cross-Platform Support
ProcHub runs natively on:
- **Windows** (amd64)
- **macOS** (Intel and Apple Silicon)
- **Linux** (amd64, arm64)

### Auto-start on Boot
Enable the application to start automatically when you log in:
- **macOS**: Uses LaunchAgent (`~/Library/LaunchAgents/`)
- **Linux**: Uses XDG Autostart (`~/.config/autostart/`)
- **Windows**: Uses Registry (`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`)

### Logging
- Rolling log files with configurable retention
- Real-time log streaming
- Separate stdout/stderr capture

## Documentation

- [Development & Build Guide](DEVELOPMENT.md) - Detailed instructions for development and building on Windows, Linux, and macOS
- [CI/CD Documentation](.github/workflows/README.md) - GitHub Actions workflow documentation
- [CI/CD 中文文档](.github/workflows/README.zh-CN.md) - GitHub Actions 工作流使用指南
- [Features](FEATURES.md) - List of application features

## Quick Start

### Prerequisites

- Go 1.23.0 or higher
- Node.js 18+ and npm
- Platform-specific build tools (see [DEVELOPMENT.md](DEVELOPMENT.md))

### Installation

```bash
# Install frontend dependencies
cd frontend
npm install
cd ..

# Install Go dependencies
go mod download

# Run in development mode
wails dev
```

## Architecture

### Backend (Go)

```
app/
├── config/           # Application configuration
│   └── config.go     # Config struct and defaults
├── i18n/             # Internationalization
│   └── locale.go     # Locale handling
├── logging/          # Log management
│   ├── rolling.go    # Rolling file logger
│   └── stream.go     # Real-time log streaming
├── process/          # Process management
│   ├── manager.go    # Process lifecycle management
│   ├── types.go      # Process types and definitions
│   ├── signal_unix.go    # Unix signal handling (macOS/Linux)
│   └── signal_windows.go # Windows process control
├── service/          # System services
│   ├── autostart_manager.go  # Cross-platform auto-start manager
│   ├── autostart_darwin.go   # macOS LaunchAgent implementation
│   ├── autostart_linux.go    # Linux XDG autostart implementation
│   └── autostart_windows.go  # Windows Registry implementation
└── store/            # Data persistence
    └── store.go      # JSON-based configuration storage
```

### API Reference

#### Process Management
- `AddProcess(def Definition)` - Register a new process
- `RemoveProcess(id string)` - Remove a process
- `UpdateProcess(id string, def Definition)` - Update process configuration
- `StartProcess(id string)` - Start a process
- `StopProcess(id string)` - Stop a process (graceful with 5s timeout)
- `RestartProcess(id string)` - Restart a process
- `ListProcesses()` - List all processes with status
- `GetProcess(id string)` - Get single process details

#### Configuration
- `GetConfig()` - Get current configuration
- `UpdateConfig(cfg AppConfig)` - Update configuration

#### Auto-start
- `GetAutoStartEnabled()` - Check if auto-start is enabled
- `SetAutoStartEnabled(enabled bool)` - Enable/disable auto-start
- `GetPlatform()` - Get current operating system

#### Utilities
- `SelectDirectory()` - Open directory selection dialog
- `GetProcessLogs(id string)` - Get logs for a process

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on http://localhost:34115. Connect
to this in your browser, and you can call your Go code from devtools.

## Building

### Build for Current Platform

```bash
wails build
```

The built application will be placed in the `build/bin` directory.

### Build for Different Platforms

See [DEVELOPMENT.md](DEVELOPMENT.md) for detailed instructions on building for Windows, Linux, and macOS, including cross-platform builds.

## CI/CD

This repository uses GitHub Actions to automatically build and package the application whenever the `main` branch is updated.

**Supported Platforms:**
- Windows (amd64)
- Linux (amd64, arm64)
- macOS (amd64/Intel, arm64/Apple Silicon)

For more information, see:
- [CI/CD Documentation](.github/workflows/README.md)
- [CI/CD 中文文档](.github/workflows/README.zh-CN.md)

## Testing

```bash
# Run Go tests
go test ./...

# Run frontend tests (if configured)
cd frontend
npm test
```

See [TEST_GUIDE.sh](TEST_GUIDE.sh) for testing instructions.

## License

This project is licensed under the MIT License.
