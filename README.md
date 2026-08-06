[English](#prochub) | [中文](README.zh-CN.md)

# ProcHub

ProcHub is a cross-platform desktop application for process management, built with Wails, Go, and Vue 3 + TypeScript. It provides an intuitive interface to manage, monitor, and control background processes across Windows, macOS, and Linux.

## Screenshot

![Process Management Home](demo/image/home.png)

### Process Management

![Process List](demo/image/home.png)

Add, remove, start, stop, and restart processes with auto-start and restart policies, plus real-time monitoring of PID, restart count, and errors.

| Add Process | Edit Process | Process Logs |
|:-----------:|:------------:|:------------:|
| ![Add Process](demo/image/process-add.png) | ![Edit Process](demo/image/process-edit.png) | ![Process Logs](demo/image/process-logs.png) |

### Advanced Settings & Environment Variables

Configure auto-start, restart policy, max retries, and custom environment variables when adding a process.

| Advanced Settings | Environment Variables |
|:-----------------:|:---------------------:|
| ![Advanced Settings](demo/image/process-add-advanced.png) | ![Environment Variables](demo/image/process-add-env.png) |

### Settings Page

Theme, language, auto-start on boot, version update, and about are managed on the settings page.

![Settings Page](demo/image/setting.png)

## Features

### Process Management
- **Add/Remove Processes**: Easily add new processes with customizable commands, arguments, and environment variables
- **Start/Stop/Restart**: Full control over process lifecycle with graceful shutdown support
- **Auto-start**: Configure processes to start automatically when the application launches
- **Restart Policies**: Support for `always`, `on_failure`, and `never` restart policies
- **Process Monitoring**: Real-time status monitoring with PID, restart count, and error tracking

### Cross-Platform Support
- **Windows** (amd64)
- **macOS** (Intel and Apple Silicon)
- **Linux** (amd64, arm64)

### Auto-start on Boot
- **macOS**: Uses LaunchAgent
- **Linux**: Uses XDG Autostart
- **Windows**: Uses Registry

### Logging
- Rolling log files with configurable retention
- Real-time log streaming
- Separate stdout/stderr capture

## Build

### Prerequisites

- Go 1.23.0 or higher
- Node.js 18+ and npm
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

### Development

```bash
# Install frontend dependencies
cd frontend && npm install && cd ..

# Install Go dependencies
go mod download

# Run in development mode
wails dev
```

### Production Build

```bash
# Build for current platform
wails build

# The built application will be in build/bin directory
```


## 💬 Join the Community

> Add friend with note "ProcHub"

| WeChat Group | QQ Group |
|:------------:|:--------:|
| <img src="https://open.tecmz.com/code_dynamic/wx" width="200" alt="WeChat Group" /> | <img src="https://open.tecmz.com/code_dynamic/qq" width="200" alt="QQ Group" /> |

## License

This project is licensed under the [Apache 2.0 License](LICENSE).

