<script lang="ts" setup>
import { ConfigProvider, message, theme } from 'ant-design-vue'
import enUS from 'ant-design-vue/es/locale/en_US'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import { computed, onMounted } from 'vue'
import { CheckVersion } from '../wailsjs/go/main/App'
import DashboardSummary from './components/layout/DashboardSummary.vue'
import ProcessList from './components/layout/ProcessList.vue'
import { trackVisit } from './services/analytics'
import { useAppStore } from './stores/app'

const appStore = useAppStore()
const currentVersion = 'v1.0.0'

// Track main page visit on mount
onMounted(() => {
  trackVisit('Main')
  
  // Auto check version after 5 seconds
  setTimeout(async () => {
    try {
      const versionInfo = await CheckVersion()
      if (versionInfo.version !== currentVersion) {
        message.info(appStore.t('settings.version.newVersion').replace('{version}', versionInfo.version))
      }
    } catch (e) {
      console.error('Auto version check failed:', e)
    }
  }, 5000)
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
      <div class="app-body">
        <DashboardSummary />
        <ProcessList />
      </div>
    </div>
  </ConfigProvider>
</template>
