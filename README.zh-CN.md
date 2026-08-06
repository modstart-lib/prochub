[English](README.md) | [中文](#prochub)

# ProcHub

ProcHub 是一个跨平台的桌面进程管理应用，使用 Wails、Go 和 Vue 3 + TypeScript 构建。它提供直观的界面来管理、监控和控制 Windows、macOS 和 Linux 上的后台进程。

## 截图预览

![进程管理主页](demo/image/home.png)

### 进程管理

![进程列表](demo/image/home.png)

支持添加、删除、启动、停止、重启进程，可配置开机自启和重启策略，实时监控 PID、重启次数和错误信息。

| 新增进程 | 编辑进程 | 进程日志 |
|:--------:|:--------:|:--------:|
| ![新增进程](demo/image/process-add.png) | ![编辑进程](demo/image/process-edit.png) | ![进程日志](demo/image/process-logs.png) |

### 新增进程 - 高级设置与环境变量

支持配置开机自启、重启策略、最大重启次数，以及自定义环境变量。

| 高级设置 | 环境变量 |
|:--------:|:--------:|
| ![高级设置](demo/image/process-add-advanced.png) | ![环境变量](demo/image/process-add-env.png) |

### 设置页

主题、语言、开机自启、版本更新与关于信息统一在设置页管理。

![设置页](demo/image/setting.png)

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

## 💬 交流沟通

> 添加好友时备注 "ProcHub"

| 微信交流群 | QQ 交流群 |
|:----------:|:---------:|
| <img src="https://open.tecmz.com/code_dynamic/wx" width="200" alt="微信交流群" /> | <img src="https://open.tecmz.com/code_dynamic/qq" width="200" alt="QQ交流群" /> |

## 许可证

本项目采用 [Apache 2.0 许可证](LICENSE)。
