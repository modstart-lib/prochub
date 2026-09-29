# Changelog

## [unreleased]

- 新增：设置页在生效数据目录非默认（`PROCHUB_DATA_ROOT` 或 `client.json` 的 `dataRoot` 覆盖）时，显示实际运行路径并支持一键打开定位到该目录
- 新增：新增 `DESIGN.md` 设计规范，统一颜色令牌、圆角、间距、字体、组件与深色模式约定
- 优化：深色模式改由 `ConfigProvider` 主题统一驱动，移除 `style.css` 中 30 余条 `.dark .ant-* !important` 双重覆盖，深色主色统一为 `#34d399`
- 优化：界面字体改为系统原生字体栈，移除 Google Fonts CDN 外链，避免离线或受限网络下加载失败
- 优化：设置页分区图标统一为 Emerald 配色（移除 indigo/green/purple/blue 混用），并修复主题切换按钮中永不生效的 indigo 覆盖
- 优化：设置页拆分复用 `SettingSection` 组件，统一图标底块、标题、间距与右对齐；「关于」改为与其他分区一致的行式布局，工单反馈弹窗改为响应式宽度并移除 iframe 负边距 hack
- 优化：进程卡片抽取为独立 `ProcessCard.vue`，进程页收敛为单一页面容器（统计区并入），页面标题字号统一
- 优化：新增/编辑进程弹窗抽取共用 `ProcessFormTabs.vue`，消除约 150 行重复模板与样式；弹窗宽度改为响应式 `min(600px, 90vw)` / `95vw`
- 优化：统一空状态组件 `AppEmptyState.vue`、统一使用 `<a-input-search />`，移除全部 `size="small"` 按钮与 `type="text"` 删除按钮，操作图标统一 `w-4 h-4` + `aria-hidden`
- 优化：日志弹窗头部与工具栏左右内边距统一，行数文案改为「共 N 行」
- 修复：修复重启策略、工单反馈、错误标签、进程计数等硬编码文案未走 i18n 的问题
- 修复：清理未使用的 Nunito 字体文件、tailwind `display/body` 字体族与 `brand-*` 颜色等死配置；`AppLogo` 补齐圆角底避免深色模式下对比度不足
- 修复：`index.html` 增加首屏主题脚本，避免深色模式启动闪白
- 测试：`tests/run.ts` 首次构建等待超时放宽至 10 分钟，并忽略 `ResizeObserver loop` 无害告警
- 截图：更新 demo 截图以反映新视觉
- 优化：端口规范化为 53090 段（53090 自动化测试端口 / 53091 前端开发端口 / 53092 截图端口 / 53093 CLI 控制接口端口），避免与其他程序调试时端口冲突。
- 新增：命令行工具（与主程序同一可执行文件，无参启动界面，带子命令时执行 CLI）：`process list/start/stop/logs`（日志支持 `--tail`/`--follow`）、`config get/set theme|language|autostart|autostart-wsl`、`status`、`version`，通过启动时写入数据目录的 `auth.json`（端口 + 令牌）与运行中的应用通信
- 新增：应用启动时在 `127.0.0.1:53093` 启动本地控制接口（携带令牌鉴权），退出时自动清理 `auth.json`
- 新增：主题改为后端配置存储（`AppConfig.theme`），新增 `SetTheme`/`SetLocale` 接口并统一配置写入路径；CLI 或外部变更主题/语言时通过事件实时同步到界面
- 优化：统一 `applyConfig` 为唯一配置写入路径（持久化 + 开机自启副作用 + 托盘语言刷新 + 界面事件通知），消除各处分散保存逻辑
- 测试：新增 `internal/control`、`internal/cli`、`internal/logging` 的控制接口与命令行测试
- 优化：配置变更改由后端统一广播全量 `config:changed` 事件（携带完整配置与变更来源），前端 Store 集中承接，CLI/托盘等外部来源改动主题、语言、开机自启后界面即刻同步
- 新增：进程列表或状态变更时广播 `processes:changed` 事件，CLI 启停进程后界面自动刷新为最新状态
- 优化：配置与进程写入全部收敛到 `applyConfig` 单一写入路径，消除 `AddProcess`、`RemoveProcess`、`UpdateProcess`、`SetProcessAutoStart`、设备 UUID 等处的旁路保存
- 修复：CLI 并发修改不同配置项时相互覆盖（读改写丢失更新），配置与进程写入增加互斥锁保护，并通过 `-race` 全量校验
- 重构：开机自启/WSL 自启状态由界面组件局部 ref 上移到 Store，移除脆弱的回调注册桥接

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
