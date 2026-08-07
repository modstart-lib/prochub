<script lang="ts" setup>
import { ConfigProvider, theme } from 'ant-design-vue'
import enUS from 'ant-design-vue/es/locale/en_US'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import { Cpu, Settings } from 'lucide-vue-next'
import { computed, getCurrentInstance, onMounted, ref } from 'vue'
import AppLogo from './components/AppLogo.vue'
import ProcessDashboardSummary from './views/Process/ProcessDashboardSummary.vue'
import ProcessList from './views/Process.vue'
import SettingsPage from './views/Setting.vue'
import { trackVisit } from './services/analytics'
import { autoCheckVersion, isAppStoreBuild } from './services/version'
import { useAppStore } from './stores/app'
import { initTestRegistry, registerNavigate, reportTestError, reportTestWarn, testActionSet } from './utils/test'
import { EventsEmit, EventsOff, EventsOn } from '../wailsjs/runtime/runtime'

const appStore = useAppStore()

// Active tab state
const activeTab = ref('processes')

// Track main page visit on mount
onMounted(async () => {
  // Initialize settings from backend
  await appStore.initSettings()

  trackVisit('Main')

  // Auto check version after 5 seconds (disabled in App Store builds)
  if (!isAppStoreBuild) {
    autoCheckVersion(5000)
  }

  // ── 自动化测试：初始化 window.__test，注册 App 级 action ─────────────────
  const _vueApp = getCurrentInstance()?.appContext.app
  initTestRegistry({
    onStartTestMode: async () => {
      // 接入 Vue 全局错误处理
      if (_vueApp) {
        const _origErrorHandler = _vueApp.config.errorHandler
        _vueApp.config.errorHandler = (err, _instance, info) => {
          const msg = err instanceof Error ? err.message : String(err)
          reportTestError(`[vue:${info}] ${msg}`)
          if (_origErrorHandler) _origErrorHandler(err, _instance, info)
          else console.error(`[vue:${info}]`, err)
        }
        _vueApp.config.warnHandler = (msg, _instance, trace) => {
          reportTestWarn(`[vue:warn] ${msg}${trace ? '\n' + trace : ''}`)
        }
      }
      // 拦截 ant-design-vue message.error / message.warning
      const { message: antMessage } = await import('ant-design-vue')
      const _origMsgError = antMessage.error.bind(antMessage)
      const _origMsgWarning = antMessage.warning.bind(antMessage)
      ;(antMessage as unknown as Record<string, unknown>).error = (...args: unknown[]) => {
        const content = typeof args[0] === 'string' ? args[0] : JSON.stringify(args[0])
        reportTestError(`[message.error] ${content}`)
        return (_origMsgError as (...a: unknown[]) => unknown)(...args)
      }
      ;(antMessage as unknown as Record<string, unknown>).warning = (...args: unknown[]) => {
        const content = typeof args[0] === 'string' ? args[0] : JSON.stringify(args[0])
        reportTestWarn(`[message.warning] ${content}`)
        return (_origMsgWarning as (...a: unknown[]) => unknown)(...args)
      }
    },
  })

  registerNavigate(async (path: string) => {
    if (path === 'processes' || path === 'settings') {
      activeTab.value = path
    }
  })

  // App 级 action
  testActionSet('App.getTitle', () => document.title)
  testActionSet('App.getActiveTab', () => activeTab.value)
  testActionSet('App.navigate', async (params: unknown) => {
    const { tab } = params as { tab: string }
    if (tab === 'processes' || tab === 'settings') {
      activeTab.value = tab
    }
    return activeTab.value
  })
  testActionSet('App.exists', (params: unknown) => {
    const { selector } = params as { selector: string }
    return !!document.querySelector(selector)
  })
  testActionSet('App.count', (params: unknown) => {
    const { selector } = params as { selector: string }
    return document.querySelectorAll(selector).length
  })
  testActionSet('App.getText', (params: unknown) => {
    const { selector } = params as { selector: string }
    return document.querySelector(selector)?.textContent?.trim() ?? null
  })
  testActionSet('App.click', (params: unknown) => {
    const { selector } = params as { selector: string }
    const el = document.querySelector(selector) as HTMLElement | null
    if (!el) throw new Error(`元素不存在: "${selector}"`)
    el.click()
    return true
  })

  // 监听来自 Go 后端的调用请求（由 HTTP /auto ui-call 转发过来）
  EventsOn('autotest:call', async (data: { id: string; name: string; params: unknown }) => {
    try {
      const result = await (window as unknown as Record<string, { callAction: (n: string, a?: unknown) => Promise<unknown> }>).__test?.callAction(data.name, data.params)
      EventsEmit('autotest:result:' + data.id, { result })
    } catch (e: unknown) {
      EventsEmit('autotest:result:' + data.id, { error: (e as Error)?.message || String(e) })
    }
  })
})

const antLocale = computed(() => {
  return appStore.locale === 'zh' ? zhCN : enUS
})

const themeConfig = computed(() => ({
  algorithm: appStore.isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
  token: {
    colorPrimary: '#10b981',
    borderRadius: 8,
    fontFamily: 'Inter, system-ui, -apple-system, BlinkMacSystemFont, sans-serif',
  },
}))
</script>

<template>
  <ConfigProvider :locale="antLocale" :theme="themeConfig">
    <div class="app-shell">
      <div class="app-layout">
        <!-- Left Sidebar with Vertical Tabs -->
        <div class="sidebar">
          <div class="logo-area">
            <AppLogo class="logo-image" />
            <span class="logo-text">ProcHub</span>
          </div>
          <div class="sidebar-tabs">
            <button
              class="tab-button"
              :class="{ active: activeTab === 'processes' }"
              @click="activeTab = 'processes'"
            >
              <Cpu :size="20" />
              <span class="tab-label">{{ appStore.t('processes.title') }}</span>
            </button>
            <button
              class="tab-button"
              :class="{ active: activeTab === 'settings' }"
              @click="activeTab = 'settings'"
            >
              <Settings :size="20" />
              <span class="tab-label">{{ appStore.t('settings.title') }}</span>
            </button>
          </div>
        </div>

        <!-- Main Content Area -->
        <div class="main-content">
          <!-- Processes View -->
          <div v-show="activeTab === 'processes'" class="content-view">
            <ProcessDashboardSummary />
            <ProcessList />
          </div>

          <!-- Settings View -->
          <div v-show="activeTab === 'settings'" class="content-view">
            <SettingsPage />
          </div>
        </div>
      </div>
    </div>
  </ConfigProvider>
</template>
