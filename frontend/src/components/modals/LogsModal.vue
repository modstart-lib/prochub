<script lang="ts" setup>
import { Button, Empty, Input, Modal, Switch, Tooltip } from 'ant-design-vue';
import { ArrowDown, Download, Filter, RefreshCw, Search, Terminal } from 'lucide-vue-next';
import { computed, nextTick, ref, watch } from 'vue';
import { SaveLogsToFile } from '../../../wailsjs/go/main/App';
import { useAppStore } from '../../stores/app';

const props = defineProps<{ 
  visible: boolean
  processId: string
  processName: string
}>()
const emit = defineEmits<{ 'update:visible': [boolean] }>()
const appStore = useAppStore()

const logs = ref<string[]>([])
const autoScroll = ref(true)
const logContainer = ref<HTMLElement | null>(null)
const searchQuery = ref('')
const showOnlyErrors = ref(false)

// 过滤日志
const filteredLogs = computed(() => {
  let result = logs.value
  
  if (showOnlyErrors.value) {
    result = result.filter(log => 
      log.toLowerCase().includes('error') || 
      log.toLowerCase().includes('stderr') ||
      log.toLowerCase().includes('failed')
    )
  }
  
  if (searchQuery.value.trim()) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(log => log.toLowerCase().includes(query))
  }
  
  return result
})

const loadLogs = async () => {
  if (!props.processId) return
  
  try {
    await appStore.loadProcessLogs(props.processId)
    logs.value = appStore.logs
    
    // Auto-scroll to bottom if enabled
    if (autoScroll.value && logContainer.value) {
      await nextTick()
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  } catch (error) {
    console.error('Failed to load logs:', error)
  }
}

const scrollToBottom = () => {
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}

const downloadLogs = async () => {
  try {
    const content = logs.value.join('\n')
    await SaveLogsToFile(props.processName, content)
  } catch (error) {
    console.error('Failed to save logs:', error)
  }
}

const getLogClass = (log: string) => {
  const lower = log.toLowerCase()
  if (lower.includes('error') || lower.includes('stderr') || lower.includes('failed')) {
    return 'log-error'
  }
  if (lower.includes('warn')) {
    return 'log-warning'
  }
  if (lower.includes('success') || lower.includes('started')) {
    return 'log-success'
  }
  return 'log-default'
}

let refreshInterval: ReturnType<typeof setInterval> | null = null

watch(
  () => props.visible,
  (next) => {
    if (next) {
      loadLogs()
      // Refresh logs every 2 seconds while modal is open
      refreshInterval = setInterval(() => {
        if (props.visible) {
          loadLogs()
        }
      }, 2000)
    } else {
      if (refreshInterval) {
        clearInterval(refreshInterval)
        refreshInterval = null
      }
      logs.value = []
      searchQuery.value = ''
      showOnlyErrors.value = false
    }
  },
)
</script>

<template>
  <Modal
    :open="props.visible"
    :title="null"
    width="900px"
    :footer="null"
    class="logs-modal"
    @cancel="emit('update:visible', false)"
  >
    <!-- 自定义头部 -->
    <div class="logs-header">
      <div class="header-title">
        <Terminal :size="20" class="title-icon" />
        <span class="title-text">{{ appStore.t('logs.title') }}</span>
        <span class="process-badge">{{ processName }}</span>
      </div>
      <div class="header-stats">
        <span class="log-count">{{ filteredLogs.length }} {{ appStore.t('logs.lines') || 'lines' }}</span>
      </div>
    </div>

    <!-- 工具栏 -->
    <div class="logs-toolbar">
      <div class="toolbar-left">
        <Input 
          v-model:value="searchQuery"
          :placeholder="appStore.t('logs.search') || 'Search logs...'"
          class="search-input"
          allow-clear
        >
          <template #prefix>
            <Search :size="14" class="search-icon" />
          </template>
        </Input>
        <Tooltip :title="appStore.t('logs.filterErrors') || 'Show only errors'">
          <Button 
            :type="showOnlyErrors ? 'primary' : 'default'"
            size="small"
            @click="showOnlyErrors = !showOnlyErrors"
          >
            <template #icon><Filter :size="14" /></template>
          </Button>
        </Tooltip>
      </div>
      <div class="toolbar-right">
        <div class="auto-scroll-toggle">
          <Switch v-model:checked="autoScroll" size="small" />
          <span class="toggle-label">{{ appStore.t('logs.autoScroll') || 'Auto-scroll' }}</span>
        </div>
        <Tooltip :title="appStore.t('logs.scrollToBottom') || 'Scroll to bottom'">
          <Button size="small" @click="scrollToBottom">
            <template #icon><ArrowDown :size="14" /></template>
          </Button>
        </Tooltip>
        <Tooltip :title="appStore.t('actions.refresh') || 'Refresh'">
          <Button size="small" @click="loadLogs">
            <template #icon><RefreshCw :size="14" /></template>
          </Button>
        </Tooltip>
        <Tooltip :title="appStore.t('logs.download') || 'Download logs'">
          <Button size="small" @click="downloadLogs" :disabled="logs.length === 0">
            <template #icon><Download :size="14" /></template>
          </Button>
        </Tooltip>
      </div>
    </div>

    <!-- 日志内容 -->
    <div ref="logContainer" class="logs-container">
      <Empty 
        v-if="filteredLogs.length === 0" 
        :description="appStore.t('logs.empty') || 'No logs available'"
        class="logs-empty"
      />
      <div v-else class="logs-content">
        <div
          v-for="(log, index) in filteredLogs"
          :key="index"
          class="log-line"
          :class="getLogClass(log)"
        >
          <span class="line-number">{{ index + 1 }}</span>
          <span class="line-content">{{ log }}</span>
        </div>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.logs-modal :deep(.ant-modal-content) {
  @apply rounded-xl overflow-hidden;
}

.logs-modal :deep(.ant-modal-body) {
  @apply p-0;
}

.logs-header {
  @apply flex items-center justify-between px-3 py-3 border-b border-slate-200 dark:border-slate-700;
}

.header-title {
  @apply flex items-center gap-2;
}

.title-icon {
  @apply text-emerald-600 dark:text-emerald-400;
}

.title-text {
  @apply text-lg font-semibold text-slate-800 dark:text-slate-200;
}

.process-badge {
  @apply ml-2 rounded-full bg-emerald-100 px-3 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-400;
}

.header-stats {
  @apply flex items-center gap-4;
}

.log-count {
  @apply text-sm text-slate-500 dark:text-slate-400;
}

.logs-toolbar {
  @apply flex items-center justify-between gap-4 px-6 py-3 border-b border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-900/50;
}

.toolbar-left {
  @apply flex items-center gap-2;
}

.search-input {
  @apply w-64;
}

.search-icon {
  @apply text-slate-400;
}

.toolbar-right {
  @apply flex items-center gap-2;
}

.auto-scroll-toggle {
  @apply flex items-center gap-2 mr-2;
}

.toggle-label {
  @apply text-xs text-slate-600 dark:text-slate-400;
}

.logs-container {
  @apply h-96 overflow-y-auto bg-slate-900;
}

.logs-empty {
  @apply py-16;
}

.logs-empty :deep(.ant-empty-description) {
  @apply text-slate-400;
}

.logs-content {
  @apply font-mono text-xs;
}

.log-line {
  @apply flex px-3 py-0.5 hover:bg-slate-800/50 transition-colors;
}

.line-number {
  @apply w-12 flex-shrink-0 text-slate-600 select-none text-right pr-4 border-r border-slate-700/50;
}

.line-content {
  @apply pl-4 text-slate-300 break-all;
}

.log-error .line-content {
  @apply text-red-400;
}

.log-warning .line-content {
  @apply text-yellow-400;
}

.log-success .line-content {
  @apply text-emerald-400;
}

.log-default .line-content {
  @apply text-slate-300;
}
</style>
