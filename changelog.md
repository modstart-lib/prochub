# Changelog

## [unreleased]

- 优化：退出与托盘交互补充诊断日志，出问题时可直接按时间线定位卡在哪一步——托盘启动/菜单就绪/图标点击/「显示界面」点击与唤起耗时、退出每一步（进程状态与耗时、完整进程快照含状态/PID/重启次数/最近错误、标记退出、投递 `WM_QUIT` 的结果或失败原因、回退 `runtime.Quit`、被阻止时的事件）、以及主窗口关闭处理（拦截隐藏 / 放行退出）与 shutdown 各阶段
- 优化：托盘日志改为「保序队列 + 单独写日志 goroutine」，既不阻塞托盘动作也不会打乱时间顺序；强制退出前的关键日志同步写入，避免进程结束时丢行，并带 300ms 看门狗防止写日志卡住退出
- 修复：托盘菜单「退出」点了没反应（长时间运行后必现）——「退出」与「显示界面」都走同一条点击通道，点击能到达但退出流程内部卡住，表现为右键菜单正常、窗口能唤起、退出却毫无反应。现在整条退出链路每一环都不再可能卡死：日志改写异步（日志磁盘 IO 卡住不会拖住退出）、进程状态查询加 1 秒超时（进程管理器卡在自己的锁上就直接放行退出）、退出动作外加 2 秒兜底（仍未结束进程则记录日志并强制结束）
- 修复：Windows 下退出改走「直投 WM_QUIT」——原先依赖 Wails 通过自定义窗口消息把退出交给主线程消息循环，该转交在应用长时间运行后可能不再被处理；现直接向主窗口线程的消息队列投递 `WM_QUIT`（与 Wails 退出等效，仍会执行 OnShutdown），找不到窗口时才回退到 `runtime.Quit`
- 修复：托盘菜单点击处理循环收到「退出」后直接 `return`——一次被阻止（仍有进程在运行）或没有真正结束进程的退出会让循环永久退出，之后右键菜单全部静默失效；现改为循环永不退出，每次点击交给独立 goroutine 执行，避免一次耗时的窗口唤起作用吞掉后续点击
- 优化：托盘交互写入系统日志（菜单就绪、退出请求、被阻止的退出及进程名、状态查询超时、兜底强制退出），便于从用户日志定位退出问题
- 新增：进程列表工具栏新增「按状态分组」开关（默认开启，选择记忆在本地）——关闭后卡片回到单一网格并按 ID 固定位置，启动/停止某个进程不会再把其他进程卡片挤到下一行
- 优化：进程卡片的换位加入过渡动画——切换分组、启停引起的分组新增/消失让后续分组整体平移，卡片在网格内移动平滑过渡，新位置卡片淡入，避免布局硬跳
- 新增：进程卡片命令超长时改为「头…尾」中间省略的单行显示（新增通用组件 `MiddleEllipsisViewer`，按容器宽度测量裁剪，悬浮显示完整命令），避免长路径换行撑高卡片
- 新增：进程生命周期审计日志——每次 spawn/exit/restart/stop 都记录 entry id、PID、命令与触发原因（`internal/process` 的 `SetLifecycleCallback`）并写入系统日志，便于定位「一个条目出现异常子进程」
- 修复：进程卡片底部布局错乱——运行信息（开机自启 / PID / 重启次数）与 5 个操作按钮挤在一起导致按钮被换行挤到下一行并左对齐、卡片高低不齐；改为「运行信息行 + #ID 与操作按钮行」，底部固定贴底、按钮统一右对齐，标题状态标签改为顶部对齐
- 修复：一个条目被拉起两个子进程——`Start` 此前仅依据 `status` 判断是否已在运行，而失败重试冷却期间 `status` 会退回到 `errored`，此时再次启动（界面「启动」按钮、CLI `process start`、重启流程等）会为同一条目再启动一个运行循环与子进程；现改以条目级的运行循环标记（`loopActive`）作为唯一判据，运行循环在任何返回路径清理该标记
- 修复：重试冷却期间「停止」无效——运行循环在每次尝试前会重置手动停止标记，导致停止后仍被重新拉起；现冷却等待可被停止中断，且 `Stop` 会等待运行循环真正退出后再返回（`RestartProcess` 的「先停后起」不再与旧循环竞争）
- 修复：进程快照 PID 与 `exec.Cmd.Start()` 写 `cmd.Process` 的数据竞争——PID 改为启动成功后于锁内发布的独立字段，`List`/`Get`/`Stop` 不再直接读取 `cmd.Process`

## v0.7.0 命令行工具与本地控制接口上线，进程管理全面升级

- 新增：`internal/config` 新增演示数据播种能力（`SeedEnabled` / `SeedDemoConfig` / `DemoProcesses`），仅在 `PROCHUB_SEED` 为真且进程列表为空时填充，保留语言/主题等既有偏好
- 新增：进程卡片新增「详情」按钮，打开只读「进程配置详情」弹窗（基础配置 / 运行状态 / 高级设置 / 环境变量），补齐运行中进程因编辑按钮被禁用而无法查看配置的缺口
- 新增：控制接口新增 `POST /control/show`（客户端 `Client.Show`），供二次启动的进程请求已运行实例恢复并聚焦主窗口
- 新增：进程新增/编辑弹窗的环境变量支持「文本导入」——粘贴标准 `.env` 文本（每行 `KEY=VALUE`，支持 `#` 注释、空行与引号包裹的值）一键导入，同名键覆盖已有值、新键追加
- 新增：运行时诊断——渲染进程错误（未捕获异常、Promise 拒绝、Vue 错误、脚本/资源加载失败）先写入本地系统日志再上报远端，并在启动与挂载阶段写入 `renderer.boot` / `renderer.mounted` 标记，便于区分「未加载」与「挂载崩溃」导致的白屏
- 新增：Go 暴露方法统一 panic 兜底（`recoverPanic`），panic 时把方法名与堆栈写入系统日志并继续向上抛出错误，避免崩溃被静默吞掉
- 新增：设置页在生效数据目录非默认（`PROCHUB_DATA_ROOT` 或 `client.json` 的 `dataRoot` 覆盖）时，显示实际运行路径并支持一键打开定位到该目录
- 新增：新增 `DESIGN.md` 设计规范，统一颜色令牌、圆角、间距、字体、组件与深色模式约定
- 新增：命令行工具（与主程序同一可执行文件，无参启动界面，带子命令时执行 CLI）：`process list/start/stop/logs`（日志支持 `--tail`/`--follow`）、`config get/set theme|language|autostart|autostart-wsl`、`wsl status/start/stop/restart`、`status`、`version`，通过启动时写入数据目录的 `auth.json`（端口 + 令牌）与运行中的应用通信
- 新增：应用启动时在 `127.0.0.1:53093` 启动本地控制接口（携带令牌鉴权），退出时自动清理 `auth.json`
- 新增：主题改为后端配置存储（`AppConfig.theme`），新增 `SetTheme`/`SetLocale` 接口并统一配置写入路径；CLI 或外部变更主题/语言时通过事件实时同步到界面
- 新增：进程列表或状态变更时广播 `processes:changed` 事件，CLI 启停进程后界面自动刷新为最新状态
- 新增：CLI 新增独立 `wsl` 子命令：`prochub wsl status|start|stop|restart`，用于查看 WSL 状态并手动启停；控制接口新增 `/control/wsl` 系列端点
- 新增：WSL 管理新增 `Stop` 能力（`wsl --shutdown`），补齐此前只有启动/重启的缺口
- 优化：新增进程详情弹窗 UI 测试，以及「运行中进程查看详情」业务逻辑测试
- 优化：进程列表改为响应式多列网格（窄屏 1 列 / 中屏 2 列 / 宽屏 3 列），并按状态分组（运行中 / 失败 / 已停止），新增带实时计数的状态筛选栏（全部 / 运行中 / 已停止 / 失败），解决进程数量达到 10 个以上时单列长滚动、难以定位的问题
- 优化：新增进程状态筛选与分组 UI 测试、以及「状态筛选 + 搜索」组合业务测试
- 优化：更新 demo 首页截图以反映进程列表多列网格与状态分组的新布局
- 优化：单实例相关路径补充系统日志（`single-instance` 组件）——前置守卫记录放行/退出原因（auth 陈旧、PID 已死、实例不可达），首实例记录收到显示请求，Wails 兜底锁记录拦截到的二次启动，便于后续定位「为何起了第二个实例」
- 优化：系统日志的目录与滚动配置收敛到 `logging.NewSystemLogger` / `logging.SystemLogDir`，供启动阶段（配置加载前）复用
- 优化：新增环境变量文本导入 UI 测试（解析去重、覆盖已有键、导入弹窗开关、编辑弹窗导入）
- 优化：Wails 框架内部日志（绑定调用错误、恢复的 panic、启动阶段日志）通过 `WailsLogger` 接入系统日志文件，与业务日志统一落盘
- 优化：设置页改为分组布局（外观 / 启动 / Windows 子系统（WSL）/ 通用）并新增 `SettingGroup` 分组标题组件，消除 WSL 项被缩进成「开机自动启动」子项的层级误导
- 优化：WSL 拆分出独立 `SettingWSL.vue`，启停按钮按状态动态显示（运行中只给「停止/重启」、已停止只给「启动/重启」），移除无意义的禁用按钮占位
- 优化：两个开机自启开关通过分组与图标区分——ProcHub 自启改用 Rocket 图标，「WSL 开机自启」归入 WSL 分组并简化文案
- 优化：深色模式改由 `ConfigProvider` 主题统一驱动，移除 `style.css` 中 30 余条 `.dark .ant-* !important` 双重覆盖，深色主色统一为 `#34d399`
- 优化：界面字体改为系统原生字体栈，移除 Google Fonts CDN 外链，避免离线或受限网络下加载失败
- 优化：设置页分区图标统一为 Emerald 配色（移除 indigo/green/purple/blue 混用），并修复主题切换按钮中永不生效的 indigo 覆盖
- 优化：设置页拆分复用 `SettingSection` 组件，统一图标底块、标题、间距与右对齐；「关于」改为与其他分区一致的行式布局，工单反馈弹窗改为响应式宽度并移除 iframe 负边距 hack
- 优化：进程卡片抽取为独立 `ProcessCard.vue`，进程页收敛为单一页面容器（统计区并入），页面标题字号统一
- 优化：新增/编辑进程弹窗抽取共用 `ProcessFormTabs.vue`，消除约 150 行重复模板与样式；弹窗宽度改为响应式 `min(600px, 90vw)` / `95vw`
- 优化：统一空状态组件 `AppEmptyState.vue`、统一使用 `<a-input-search />`，移除全部 `size="small"` 按钮与 `type="text"` 删除按钮，操作图标统一 `w-4 h-4` + `aria-hidden`
- 优化：日志弹窗头部与工具栏左右内边距统一，行数文案改为「共 N 行」
- 优化：`tests/run.ts` 首次构建等待超时放宽至 10 分钟，并忽略 `ResizeObserver loop` 无害告警
- 优化：更新 demo 截图以反映新视觉
- 优化：端口规范化为 53090 段（53090 自动化测试端口 / 53091 前端开发端口 / 53092 截图端口 / 53093 CLI 控制接口端口），避免与其他程序调试时端口冲突。
- 优化：统一 `applyConfig` 为唯一配置写入路径（持久化 + 开机自启副作用 + 托盘语言刷新 + 界面事件通知），消除各处分散保存逻辑
- 优化：新增 `internal/control`、`internal/cli`、`internal/logging` 的控制接口与命令行测试
- 优化：配置变更改由后端统一广播全量 `config:changed` 事件（携带完整配置与变更来源），前端 Store 集中承接，CLI/托盘等外部来源改动主题、语言、开机自启后界面即刻同步
- 优化：配置与进程写入全部收敛到 `applyConfig` 单一写入路径，消除 `AddProcess`、`RemoveProcess`、`UpdateProcess`、`SetProcessAutoStart`、设备 UUID 等处的旁路保存
- 优化：开机自启/WSL 自启状态由界面组件局部 ref 上移到 Store，移除脆弱的回调注册桥接
- 优化：解除 WSL 与「开机自启」的相互依赖——`AutoStartWSL` 不再需要应用开机自启为前置条件，`startWSLOnBoot` 移除对 `IsEnabled` 的检查
- 优化：设置页 WSL 拆为独立分区（运行状态 + 启动/停止/重启按钮 + 开机自启开关），仅在 Windows 显示，与应用开机自启状态完全解耦；移除 `wsl-inactive` 置灰与 `requiresAutoStart` 提示
- 修复：Windows 下偶发双击 exe 起第二份进程并卡死（无窗口、无托盘图标，只能到任务管理器结束）——Wails v2 的 Windows 单实例锁在 `CreateMutex` 返回非「已存在」错误（如以管理员/普通身份混用时权限不一致）或首实例隐藏窗口尚未就绪时会直接放行新实例，而新实例与首实例共用同一个 WebView2 用户数据目录，会在 WebView2 初始化中因 `ERROR_INVALID_STATE` 永久卡在自旋消息循环；现在 `main()` 在 `wails.Run` 之前增加进程级前置守卫：读取数据目录 `auth.json`、校验 PID 存活并向首实例 `POST /control/show` 让其显示窗口后直接 `os.Exit(0)`，从根本上不让第二个实例走到 WebView2，Wails 单实例锁保留为兜底
- 修复：Windows 托盘图标运行一段时间后失去响应（无法双击唤起窗口、无法右键菜单、无法退出）——托盘消息循环由 `go trayManager.Run()` 在新 goroutine 中运行，而 `fyne.io/systray` 用 `GetMessageW` 在调用线程上抽取消息、Windows 又把窗口消息投递到创建窗口的线程，goroutine 被调度器迁移到其它 OS 线程后即抽错队列、消息再也无法分发；现于 `Run()` 的 Windows 分支调用 `runtime.LockOSThread()` 把消息循环固定在拥有窗口的线程上，并把托盘左键回调改到独立 goroutine 执行，避免 `ShowWindow`/`SetForegroundWindow` 等跨线程调用阻塞消息循环
- 修复：Windows 下守护启动的控制台子进程日志中文乱码——子进程输出被重定向到管道时按系统 ANSI 代码页（简体中文为 CP936/GBK）而非 UTF-8 编码，现统一解码为 UTF-8 后再落盘与推送（`internal/platform` 新增 `DecodeOutput`，按代码页转码 UTF-8 直通）；同时修复系统信息中 `osVersion` 同类乱码（如「版本」显示为「�汾」）
- 修复：设置页「关于」GitHub 按钮图标由 `Globe`（易误解为官网）改为 `Github`
- 修复：修复重启策略、工单反馈、错误标签、进程计数等硬编码文案未走 i18n 的问题
- 修复：清理未使用的 Nunito 字体文件、tailwind `display/body` 字体族与 `brand-*` 颜色等死配置；`AppLogo` 补齐圆角底避免深色模式下对比度不足
- 修复：`index.html` 增加首屏主题脚本，避免深色模式启动闪白
- 修复：CLI 并发修改不同配置项时相互覆盖（读改写丢失更新），配置与进程写入增加互斥锁保护，并通过 `-race` 全量校验

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
