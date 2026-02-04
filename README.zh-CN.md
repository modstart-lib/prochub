[English](README.md) | [中文](#prochub)

# ProcHub

ProcHub 是一个跨平台的桌面进程管理应用，使用 Wails、Go 和 Vue 3 + TypeScript 构建。它提供直观的界面来管理、监控和控制 Windows、macOS 和 Linux 上的后台进程。

## 截图预览

![](https://ms-assets.modstart.com/data/image/2026/02/04/6139_s17d_1578.png)

![](https://ms-assets.modstart.com/data/image/2026/02/04/6163_fb7s_9127.png)

![](https://ms-assets.modstart.com/data/image/2026/02/04/6184_nlto_7268.png)

![](https://ms-assets.modstart.com/data/image/2026/02/04/6211_sf2x_6063.png)

## 功能特性

### 进程管理
- **添加/删除进程**：轻松添加新进程，支持自定义命令、参数和环境变量
- **启动/停止/重启**：完整的进程生命周期控制，支持优雅关闭
- **自动启动**：配置进程在应用启动时自动运行
- **重启策略**：支持 `always`（始终）、`on_failure`（失败时）和 `never`（从不）重启策略
- **进程监控**：实时状态监控，包括 PID、重启次数和错误追踪

### 跨平台支持
- **Windows** (amd64)
- **macOS** (Intel 和 Apple Silicon)
- **Linux** (amd64, arm64)

### 开机自启
- **macOS**：使用 LaunchAgent
- **Linux**：使用 XDG Autostart
- **Windows**：使用注册表

### 日志功能
- 滚动日志文件，支持可配置的保留策略
- 实时日志流
- 分离的 stdout/stderr 捕获

## 构建

### 环境要求

- Go 1.23.0 或更高版本
- Node.js 18+ 和 npm
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

### 开发模式

```bash
# 安装前端依赖
cd frontend && npm install && cd ..

# 安装 Go 依赖
go mod download

# 运行开发模式
wails dev
```

### 生产构建

```bash
# 为当前平台构建
wails build

# 构建的应用程序将在 build/bin 目录中
```

## 许可证

本项目采用 [Apache 2.0 许可证](LICENSE)。
