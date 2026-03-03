# Changelog

## [Unreleased]

### Fixes
- Update import paths and component references after moving and renaming view files / 在移动和重命名视图文件后修复导入路径和组件引用

### Improvements
- Remove `com.apple.security.network.server` entitlement from macOS entitlements / 从 macOS entitlements 中移除 `com.apple.security.network.server` 权限
- Disable browser spellcheck on `<body>` element by setting `spellcheck="false"` / 通过在 `<body>` 元素上设置 `spellcheck="false"` 禁用浏览器拼写检查

### Others
- Add new asset `res/logo.svg.black.svg` / 新增资源文件 `res/logo.svg.black.svg`
- Bump version to v0.5.9 / 版本升级至 v0.5.9

## [v0.5.8] - 2026-02-24

### Features
- Add `isAppStoreBuild` flag driven by `VITE_APPSTORE_BUILD` env variable to detect App Store distribution builds / 新增 `isAppStoreBuild` 标识，通过 `VITE_APPSTORE_BUILD` 环境变量检测是否为 App Store 发布构建
- Hide version check button and disable auto version check in App Store builds / 在 App Store 构建中隐藏版本检测按钮并禁用启动时自动版本检测

### Improvements
- App Store CI workflow now passes `VITE_APPSTORE_BUILD=true` to the Wails build step automatically / App Store CI 工作流在 Wails 构建步骤中自动注入 `VITE_APPSTORE_BUILD=true` 环境变量

### Others
- Bump version to v0.5.8 / 版本升级至 v0.5.8
