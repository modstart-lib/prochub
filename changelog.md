# Changelog

## v0.6.0 Windows 托盘交互优化，WSL 自动启动，数据目录配置增强

- 新增：Windows 下单击托盘图标直接显示主界面，右键才弹出托盘菜单
- 新增：托盘退出时若仍有进程在运行则阻止退出，自动显示窗口并在界面弹出提示（列出运行中的进程名），需先手动停止全部进程后才能退出；`QuitApp` 采用相同策略
- 新增：Windows 下设置页「开机自动启动」下方新增「自动启动 WSL」子选项，仅当开启开机自动启动时生效；开启后会显示 WSL 运行状态（运行中/已停止/未安装）并支持「开启」「重启」操作，应用随开机启动时自动拉起 WSL
- 新增：新增 `make update-version VERSION=x.y.z` 命令（`scripts/update-version.sh`），一键同步更新 `app.go`、根/前端 `package.json`、两个 `package-lock.json`、`tests/screenshot.ts` 中的版本号，支持 `x.y.z` 及 `x.y.z-beta` 格式并校验非法输入
- 新增：数据目录支持 `PROCHUB_DATA_ROOT` 环境变量覆盖（优先级最高，支持 `~/` 展开），并新增 `~/.prochub/client.json` 的 `dataRoot` 字段（默认 `~/.prochub/data`）作为第二优先级；正式安装版可通过 macOS Info.plist 的 `LSEnvironment` 注入，实现正式使用数据与开发测试完全隔离
- 新增：新增跨平台数据目录解析模块（`internal/platform/appdir`），统一将配置与日志存储至 `~/.prochub` 并自动创建目录
- 优化：数据目录解析逻辑收敛至平台模块，兼容 macOS/Linux/Windows（含 HOME 环境变量缺失时的兜底处理）
- 修复：Windows 下被隐藏的窗口无法恢复显示：Wails 的 `WindowShow`/`WindowUnminimise` 仅调用 `SW_SHOW`，对 `SW_HIDE` 隐藏的窗口无效，新增 `_windows` 平台实现按窗口类名与标题定位 HWND 并调用 `SW_RESTORE`；同时统一为 `platform.ShowMainWindow`（应用于托盘显示、二次启动、退出受阻等场景）
- 修复：Windows 下托盘「退出」被运行中进程阻止时，若界面此前已隐藏则先恢复并显示界面，再弹出提示，避免提示弹窗用户看不到
- 修复：托盘菜单「退出」无法真正退出应用（图标消失但进程残留）：新增 quitting 标志区分「关闭窗口=隐藏」与「真正退出」，退出时 `OnBeforeClose` 不再拦截关闭，应用可正常退出
- 修复：Windows 下调用 `wsl.exe` 不再弹出黑色控制台窗口（`CREATE_NO_WINDOW`）；启动 WSL 改为用登录 shell（`/bin/bash -l -i`）拉起并持有常驻，确保 `/etc/profile`、`~/.profile`、`~/.bashrc` 生效（环境与命令行 `wsl` 一致），并新增「启动中」状态以覆盖 WSL 冷启动耗时较长的情况
- 修复：`make dev-seed-test` 进程清理由 `pkill -f ProcHub` 改为精确匹配 `build/bin/ProcHub`，避免误杀已安装的正式版 ProcHub.app；测试启动前强制清除 `PROCHUB_DATA_ROOT`，确保种子数据只写入默认目录
- 修复：`ProcessEditModal.vue` 使用 `FileSearch` 图标但未导入（测试拦截 Vue warn 时发现）

## v0.5.0 App Store 构建支持，组件重构更易维护

- 新增：新增 `isAppStoreBuild` 标识，通过 `VITE_APPSTORE_BUILD` 环境变量检测是否为 App Store 发布构建
- 新增：在 App Store 构建中隐藏版本检测按钮并禁用启动时自动版本检测
- 优化：将 `Setting.vue` 拆分为独立子组件：`SettingTheme`、`SettingLanguage`、`SettingAutoStart`、`SettingVersion`、`SettingAbout`，提升代码可维护性
- 优化：重构进程模态框组件：将 `AddModal`、`EditModal`、`LogsModal` 重命名为 `ProcessAddModal`、`ProcessEditModal`、`ProcessLogsModal`，并新增 `ProcessDashboardSummary` 组件
- 优化：从 macOS entitlements 中移除 `com.apple.security.network.server` 权限
- 优化：通过在 `<body>` 元素上设置 `spellcheck="false"` 禁用浏览器拼写检查
- 优化：App Store CI 工作流在 Wails 构建步骤中自动注入 `VITE_APPSTORE_BUILD=true` 环境变量
- 修复：修复 Windows 托盘图标不显示：Windows 下改用 `systray.Run`，使托盘窗口与其消息循环在同一系统线程运行
- 修复：修复进程卡片上无法点击的开机自启开关：移除硬编码的 `disabled`，新增 `SetProcessAutoStart` 后端接口，切换开关不会停止正在运行的进程
- 修复：修复在移动和重命名视图文件后导入路径和组件引用错误

## v0.4.0

- v0.4.0 发布

## v0.3.0

- v0.3.0 发布

## v0.2.0

- v0.2.0 发布

## v0.1.0

- v0.1.0 发布
