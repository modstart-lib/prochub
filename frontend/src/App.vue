<script lang="ts" setup>
import { ConfigProvider, theme } from 'ant-design-vue'
import enUS from 'ant-design-vue/es/locale/en_US'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import { Cpu, Settings } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import DashboardSummary from './components/layout/DashboardSummary.vue'
import ProcessList from './components/layout/ProcessList.vue'
import SettingsPage from './components/layout/SettingsPage.vue'
import { trackVisit } from './services/analytics'
import { autoCheckVersion } from './services/version'
import { useAppStore } from './stores/app'

const appStore = useAppStore()

// Track main page visit on mount
onMounted(() => {
  trackVisit('Main')
  
  // Auto check version after 5 seconds
  autoCheckVersion(5000)
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

// Active tab state
const activeTab = ref('processes')
</script>

<template>
  <ConfigProvider :locale="antLocale" :theme="themeConfig">
    <div class="app-shell">
      <div class="app-layout">
        <!-- Left Sidebar with Vertical Tabs -->
        <div class="sidebar">
          <div class="sidebar-tabs">
            <Tooltip :title="appStore.t('processes.title')" placement="right">
              <button 
                class="tab-button"
                :class="{ active: activeTab === 'processes' }"
                @click="activeTab = 'processes'"
              >
                <Cpu :size="22" />
              </button>
            </Tooltip>
            <Tooltip :title="appStore.t('settings.title')" placement="right">
              <button 
                class="tab-button"
                :class="{ active: activeTab === 'settings' }"
                @click="activeTab = 'settings'"
              >
                <Settings :size="22" />
              </button>
            </Tooltip>
          </div>
        </div>

        <!-- Main Content Area -->
        <div class="main-content">
          <!-- Processes View -->
          <div v-show="activeTab === 'processes'" class="content-view">
            <DashboardSummary />
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
