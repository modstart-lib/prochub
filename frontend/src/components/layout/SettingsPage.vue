<script lang="ts" setup>
import { Button, Card, Divider, message, Modal, RadioButton, RadioGroup, Select, Switch } from 'ant-design-vue';
import { Globe, Info, Languages, MessageSquare, Moon, Power, RefreshCw, Sun } from 'lucide-vue-next';
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { GetAppConfig, GetAutoStartEnabled, GetPlatform, GetProcessLogs, GetSystemLogs, GetSystemVersion, ListProcesses, SetAutoStartEnabled } from '../../../wailsjs/go/main/App';
import { trackVisit } from '../../services/analytics';
import { checkVersionAndPrompt, getAppVersion } from '../../services/version';
import { useAppStore } from '../../stores/app';

const appStore = useAppStore()

// Track visit on mount
onMounted(() => {
  trackVisit('Settings')
})

const themeMode = computed({
  get: () => (appStore.isDark ? 'dark' : 'light'),
  set: (value: 'light' | 'dark') => {
    appStore.setTheme(value === 'dark')
  },
})

const locale = computed({
  get: () => appStore.locale,
  set: (value: 'zh' | 'en') => {
    appStore.setLocale(value)
  },
})

const languageOptions = [
  { value: 'zh', label: '中文' },
  { value: 'en', label: 'English' },
]

const autoStart = ref(false)
const appVersion = ref('')
const feedbackUrl = ref('')
const showFeedbackModal = ref(false)

// Load appVersion and config on mount
onMounted(async () => {
  appVersion.value = await getAppVersion()
  try {
    const config = await GetAppConfig()
    if (config && config.feedbackUrl) {
      feedbackUrl.value = config.feedbackUrl
    }
  } catch (e) {
    console.error('Failed to load app config:', e)
  }
  
  try {
    autoStart.value = await GetAutoStartEnabled()
  } catch (e) {
    console.error('Failed to load auto-start status:', e)
  }
})

const toggleAutoStart = async (checked: boolean | string | number) => {
  try {
    await SetAutoStartEnabled(!!checked)
    autoStart.value = !!checked
  } catch (e) {
    console.error('Failed to update auto-start:', e)
    autoStart.value = !checked
  }
}

// Version check state
const versionChecking = ref(false)

const handleCheckVersion = async () => {
  versionChecking.value = true
  try {
    await checkVersionAndPrompt({ showLatestMessage: true, showErrorMessage: true })
  } finally {
    versionChecking.value = false
  }
}

const copyGithubLink = async () => {
  try {
    await navigator.clipboard.writeText('https://github.com/modstart-lib/prochub')
    message.success(appStore.t('settings.linkCopied'))
  } catch (e) {
    console.error('Failed to copy link:', e)
    message.error(appStore.t('settings.linkCopyFailed'))
  }
}


const handleFeedbackMessage = async (event: MessageEvent) => {
  if (!event.data || !event.data.type) return

  const type = event.data.type

  if (type === 'FeedbackTicket:env') {
    try {
      const version = await getAppVersion()
      const processes = await ListProcesses()
      const platform = await GetPlatform()
      const systemVersion = await GetSystemVersion()
      
      const envData = {
        version,
        platform,
        systemVersion,
        processes: processes.map((p: any) => ({
          name: p.definition.name,
          status: p.status,
          restarts: p.restarts
        }))
      }
      
      if (event.source) {
        (event.source as Window).postMessage({
          type: 'FeedbackTicket:env',
          data: envData
        }, '*')
      }
    } catch (e) {
      console.error('Failed to collect env:', e)
    }
  } else if (type === 'FeedbackTicket:log') {
    try {
      const processes = await ListProcesses()
      let allLogs = ''
      
      // Collect logs from all processes
      for (const p of processes) {
        const logs = await GetProcessLogs(p.definition.id)
        if (logs && logs.length > 0) {
          allLogs += `\n=== Process: ${p.definition.name} ===\n`
          allLogs += logs.map((l: any) => `[${l.timestamp}] ${l.stream}: ${l.line}`).join('\n')
          allLogs += '\n'
        }
      }
      
      // Collect system logs
      const systemLogs = await GetSystemLogs()
      if (systemLogs) {
        allLogs += '\n' + systemLogs
      }
      
      const now = new Date()
      const startTime = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString()
      const endTime = now.toISOString()
      
      if (event.source) {
        (event.source as Window).postMessage({
          type: 'FeedbackTicket:log',
          data: {
            logs: allLogs,
            startTime,
            endTime
          }
        }, '*')
      }
    } catch (e) {
      console.error('Failed to collect logs:', e)
    }
  }
}

// Register message listener
onMounted(() => {
  window.addEventListener('message', handleFeedbackMessage)
})

onUnmounted(() => {
  window.removeEventListener('message', handleFeedbackMessage)
})
</script>

<template>
  <div class="settings-page">
    <div class="settings-header">
      <h2 class="settings-title">{{ appStore.t('settings.title') }}</h2>
    </div>

    <div class="settings-content">
      <!-- 主题设置 -->
      <div class="setting-section">
        <div class="section-header">
          <div class="section-icon theme-icon">
            <Sun v-if="!appStore.isDark" :size="18" />
            <Moon v-else :size="18" />
          </div>
          <div class="section-info">
            <h3 class="section-title">{{ appStore.t('settings.theme.title') }}</h3>
            <p class="section-desc">{{ appStore.t('settings.theme.desc') }}</p>
          </div>
        </div>
        <div class="section-control">
          <RadioGroup v-model:value="themeMode" button-style="solid">
            <RadioButton value="light" class="theme-button">
              <Sun :size="14" class="button-icon" />
              {{ appStore.t('settings.theme.light') }}
            </RadioButton>
            <RadioButton value="dark" class="theme-button">
              <Moon :size="14" class="button-icon" />
              {{ appStore.t('settings.theme.dark') }}
            </RadioButton>
          </RadioGroup>
        </div>
      </div>

      <Divider class="section-divider" />

      <!-- 语言设置 -->
      <div class="setting-section">
        <div class="section-header">
          <div class="section-icon language-icon">
            <Languages :size="18" />
          </div>
          <div class="section-info">
            <h3 class="section-title">{{ appStore.t('settings.language') }}</h3>
            <p class="section-desc">{{ appStore.t('settings.languageDesc') }}</p>
          </div>
        </div>
        <div class="section-control">
          <Select 
            v-model:value="locale" 
            :options="languageOptions"
            class="language-select"
            size="middle"
          >
            <template #suffixIcon>
              <Globe :size="14" />
            </template>
          </Select>
        </div>
      </div>

      <Divider class="section-divider" />

      <!-- 开机自动启动 -->
      <div class="setting-section">
        <div class="section-header">
          <div class="section-icon autostart-icon">
            <Power :size="18" />
          </div>
          <div class="section-info">
            <h3 class="section-title">{{ appStore.t('settings.autoStart.title') }}</h3>
            <p class="section-desc">{{ appStore.t('settings.autoStart.desc') }}</p>
          </div>
        </div>
        <div class="section-control">
          <Switch 
            :checked="autoStart" 
            @change="toggleAutoStart"
          />
        </div>
      </div>

      <Divider class="section-divider" />

      <!-- 版本检测 -->
      <div class="setting-section">
        <div class="section-header">
          <div class="section-icon version-icon">
            <RefreshCw :size="18" />
          </div>
          <div class="section-info">
            <h3 class="section-title">{{ appStore.t('settings.version.title') }}</h3>
            <p class="section-desc">{{ appStore.t('settings.version.desc') }}</p>
          </div>
        </div>
        <div class="section-control">
          <div class="version-control">
            <span class="current-version">{{ appStore.t('settings.version.currentVersion') }}: {{ appVersion }}</span>
            <Button 
              type="primary" 
              size="small"
              :loading="versionChecking"
              @click="handleCheckVersion"
            >
              <template #icon>
                <RefreshCw :size="14" v-if="!versionChecking" />
              </template>
              {{ versionChecking ? appStore.t('settings.version.checking') : appStore.t('settings.version.checkUpdate') }}
            </Button>
          </div>
        </div>
      </div>

      <Divider class="section-divider" />

      <!-- 关于 -->
      <Card class="about-card" :bordered="false">
        <template #title>
          <div class="about-header">
            <Info :size="16" class="about-icon" />
            <span>{{ appStore.t('settings.about') }}</span>
          </div>
        </template>
        <div class="about-content">
          <p class="about-text">{{ appStore.t('settings.aboutDesc') }}</p>
          <div class="about-meta">
            <span 
              class="github-link"
              @click="copyGithubLink"
              role="button"
              tabindex="0"
            >
              <Globe :size="14" />
              github.com/modstart-lib/prochub
            </span>
            <Button 
              v-if="feedbackUrl"
              type="primary" 
              size="small"
              class="feedback-btn ml-auto"
              @click="showFeedbackModal = true"
            >
              <template #icon>
                <MessageSquare :size="14" />
              </template>
              工单反馈
            </Button>
          </div>
        </div>
      </Card>
    </div>

    <!-- Feedback Modal -->
    <Modal
      v-model:open="showFeedbackModal"
      title="工单反馈"
      :footer="null"
      width="600px"
      style="top: 20px"
      :bodyStyle="{ padding: 0, height: '70vh', overflow: 'visible' }"
    >
      <div style="width:calc(48px + 100%); height:calc(20px + 100%);overflow:hidden;border-radius:0 0 8px 8px;margin:0px 24px -20px -24px;">
        <iframe 
          v-if="feedbackUrl" 
          :src="feedbackUrl" 
          style="width: 100%; height: 100%;"
        ></iframe>
      </div>
    </Modal>
  </div>
</template>

<style scoped>
.settings-page {
  @apply flex flex-col rounded-xl border border-slate-200/60 bg-white/80 backdrop-blur-sm dark:border-slate-700/60 dark:bg-slate-800/80 p-6;
}

.settings-header {
  @apply pb-4 border-b border-slate-200 dark:border-slate-700;
}

.settings-title {
  @apply text-xl font-bold text-slate-800 dark:text-slate-200;
}

.settings-content {
  @apply pt-6;
}

.setting-section {
  @apply flex flex-row items-center justify-between gap-4;
}

.section-header {
  @apply flex items-center gap-3;
}

.section-icon {
  @apply flex h-10 w-10 items-center justify-center rounded-lg;
}

.theme-icon {
  @apply bg-amber-100 text-amber-600 dark:bg-amber-900/50 dark:text-amber-400;
}

.language-icon {
  @apply bg-blue-100 text-blue-600 dark:bg-blue-900/50 dark:text-blue-400;
}

.autostart-icon {
  @apply bg-green-100 text-green-600 dark:bg-green-900/50 dark:text-green-400;
}

.version-icon {
  @apply bg-purple-100 text-purple-600 dark:bg-purple-900/50 dark:text-purple-400;
}

.section-info {
  @apply flex flex-col;
}

.section-title {
  @apply text-sm font-semibold text-slate-800 dark:text-slate-200;
}

.section-desc {
  @apply text-xs text-slate-500 dark:text-slate-400;
}

.section-control {
  @apply flex items-center;
}

.theme-button {
  @apply flex items-center gap-1.5;
}

.button-icon {
  @apply -ml-0.5;
}

.language-select {
  @apply w-40;
}

.version-control {
  @apply flex items-center gap-3;
}

.current-version {
  @apply text-xs text-slate-500 dark:text-slate-400;
}

.section-divider {
  @apply my-4;
}

.about-card {
  @apply rounded-xl bg-slate-50 dark:bg-slate-800/50;
}

.about-card :deep(.ant-card-head) {
  @apply border-b-0 pb-0;
}

.about-card :deep(.ant-card-body) {
  @apply pt-2;
}

.about-header {
  @apply flex items-center gap-2 text-sm font-semibold text-slate-700 dark:text-slate-300;
}

.about-icon {
  @apply text-slate-500;
}

.about-content {
  @apply flex flex-col gap-3;
}

.about-text {
  @apply text-sm text-slate-600 dark:text-slate-400 leading-relaxed;
}

.about-meta {
  @apply flex items-center gap-2;
}

.github-link {
  @apply flex items-center gap-1.5 text-xs text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300 transition-colors cursor-pointer;
}

.feedback-btn {
  @apply flex items-center gap-1.5 text-xs;
}

/* 主题切换按钮样式修复 */
.section-control :deep(.ant-radio-group) {
  @apply flex;
}

.section-control :deep(.ant-radio-button-wrapper) {
  @apply flex items-center gap-1.5 border-slate-300 dark:border-slate-600;
}

.section-control :deep(.ant-radio-button-wrapper > span) {
  @apply flex items-center gap-1.5;
}

.section-control :deep(.ant-radio-button-wrapper:first-child) {
  @apply rounded-l-lg;
}

.section-control :deep(.ant-radio-button-wrapper:last-child) {
  @apply rounded-r-lg;
}

.section-control :deep(.ant-radio-button-wrapper-checked) {
  @apply bg-indigo-500 border-indigo-500 text-white;
}

.section-control :deep(.ant-radio-button-wrapper-checked:hover) {
  @apply bg-indigo-600 border-indigo-600;
}

.section-control :deep(.ant-radio-button-wrapper:not(.ant-radio-button-wrapper-checked)) {
  @apply bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300;
}

.section-control :deep(.ant-radio-button-wrapper:not(.ant-radio-button-wrapper-checked):hover) {
  @apply text-indigo-500 dark:text-indigo-400;
}
</style>
