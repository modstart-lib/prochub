# Changelog

## [Unreleased]

- 新增：新增跨平台数据目录解析模块（`internal/platform/appdir`），统一将配置与日志存储至 `~/.prochub` 并自动创建目录
- 优化：数据目录解析逻辑收敛至平台模块，兼容 macOS/Linux/Windows（含 HOME 环境变量缺失时的兜底处理）

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
