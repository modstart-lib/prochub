<script lang="ts" setup>
import { ConfigProvider, theme } from 'ant-design-vue'
import enUS from 'ant-design-vue/es/locale/en_US'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import { computed, onMounted } from 'vue'
import DashboardSummary from './components/layout/DashboardSummary.vue'
import ProcessList from './components/layout/ProcessList.vue'
import { trackVisit } from './services/analytics'
import { useAppStore } from './stores/app'

const appStore = useAppStore()

// Track main page visit on mount
onMounted(() => {
  trackVisit('Main')
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
