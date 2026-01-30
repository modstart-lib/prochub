<script lang="ts" setup>
import { Button, Input, message, Spin, Switch, Tag, Tooltip } from 'ant-design-vue'
import { Cpu, FileText, Hash, Pencil, Play, Plus, RefreshCw, RotateCw, Settings, Square } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import type { ProcessItem } from '../../stores/app'
import { useAppStore } from '../../stores/app'
import AddProcessModal from '../modals/AddProcessModal.vue'
import EditProcessModal from '../modals/EditProcessModal.vue'
import LogsModal from '../modals/LogsModal.vue'
import SettingsModal from '../modals/SettingsModal.vue'

const appStore = useAppStore()
const showModal = ref(false)
const showEditModal = ref(false)
const showSettingsModal = ref(false)
const showLogsModal = ref(false)
const editingProcess = ref<ProcessItem | null>(null)
const logsProcess = ref<ProcessItem | null>(null)
const searchQuery = ref('')
const loadingProcessId = ref<string | null>(null)

// 过滤并按ID排序进程列表（确保顺序固定）
const filteredProcesses = computed(() => {
  let list = [...appStore.processes]
  // 按ID排序，确保列表顺序固定
  list.sort((a, b) => a.id.localeCompare(b.id))
  if (!searchQuery.value.trim()) {
    return list
  }
  const query = searchQuery.value.toLowerCase()
  return list.filter(
    (p) => p.name.toLowerCase().includes(query) || p.command.toLowerCase().includes(query)
  )
})

// Load processes on mount
onMounted(() => {
  appStore.loadProcesses()
  // Auto-refresh every 3 seconds
  setInterval(() => {
    appStore.loadProcesses()
  }, 3000)
})

const getStatusConfig = (status: string) => {
  if (status === 'running' || status === 'starting') {
    return { color: 'success', text: 'running', dotClass: 'status-dot-running' }
  }
  if (status === 'errored') {
    return { color: 'error', text: 'failed', dotClass: 'status-dot-failed' }
  }
  return { color: 'default', text: 'stopped', dotClass: 'status-dot-stopped' }
}

const openEditModal = (process: ProcessItem) => {
  editingProcess.value = process
  showEditModal.value = true
}

const openSettings = () => {
  showSettingsModal.value = true
}

const handleStart = async (process: ProcessItem) => {
  loadingProcessId.value = process.id
  try {
    if (process.status === 'running' || process.status === 'starting') {
      await appStore.stopProcess(process.id)
      message.success(appStore.t('messages.processStopped'))
    } else {
      await appStore.startProcess(process.id)
      message.success(appStore.t('messages.processStarted'))
    }
  } catch (error) {
    message.error(appStore.t('messages.operationFailed'))
  } finally {
    loadingProcessId.value = null
  }
}

const handleRestart = async (process: ProcessItem) => {
  loadingProcessId.value = process.id
  try {
    await appStore.restartProcess(process.id)
    message.success(appStore.t('messages.processRestarted'))
  } catch (error) {
    message.error(appStore.t('messages.operationFailed'))
  } finally {
    loadingProcessId.value = null
  }
}

const handleLogs = async (process: ProcessItem) => {
  logsProcess.value = process
  showLogsModal.value = true
}
</script>

<template>
  <section class="process-list-container">
    <!-- 头部 -->
    <div class="process-header">
      <div class="header-left">
        <h2 class="header-title">
          <Cpu :size="20" class="title-icon" />
          {{ appStore.t('processes.title') }}
        </h2>
        <span class="process-count">{{ filteredProcesses.length }} {{ appStore.t('processes.title') }}</span>
      </div>
      <div class="header-actions">
        <Input 
          v-model:value="searchQuery"
          :placeholder="appStore.t('actions.search') || 'Search...'"
          class="search-input"
          allow-clear
        />
        <Button type="primary" @click="showModal = true">
          <template #icon><Plus :size="16" /></template>
          {{ appStore.t('actions.addProcess') }}
        </Button>
        <Tooltip :title="appStore.t('settings.title')">
          <Button @click="openSettings">
            <template #icon><Settings :size="16" /></template>
          </Button>
        </Tooltip>
      </div>
    </div>

    <!-- 进程列表 -->
    <div class="process-grid">
      <div v-if="filteredProcesses.length === 0" class="empty-state">
        <Cpu :size="48" class="empty-icon" />
        <p class="empty-text">{{ appStore.t('processes.empty') || 'No processes found' }}</p>
      </div>

      <div
        v-for="process in filteredProcesses"
        :key="process.id"
        class="process-card"
        :class="{ 'process-card-running': process.status === 'running' || process.status === 'starting' }"
      >
        <Spin :spinning="loadingProcessId === process.id" size="small">
          <!-- 卡片头部 -->
          <div class="card-header">
            <div class="card-title-section">
              <div class="status-indicator" :class="getStatusConfig(process.status).dotClass"></div>
              <h3 class="card-title">{{ process.name }}</h3>
            </div>
            <Tag :color="getStatusConfig(process.status).color" class="status-tag">
              {{ appStore.t(`processes.status.${process.status}`) }}
            </Tag>
          </div>

          <!-- 命令信息 -->
          <div class="card-command">
            <code class="command-text">{{ process.command }}</code>
          </div>

          <!-- 错误信息 -->
          <div v-if="process.lastError" class="card-error">
            <span class="error-label">Error:</span>
            <span class="error-text">{{ process.lastError }}</span>
          </div>

          <!-- 元信息和操作按钮 -->
          <div class="card-meta">
            <div class="meta-left">
              <div class="meta-item">
                <Switch 
                  :checked="process.autoStart" 
                  size="small" 
                  disabled 
                />
                <span class="meta-label">{{ appStore.t('processes.autoStart') }}</span>
              </div>
              <div class="meta-item" v-if="process.pid > 0">
                <Hash :size="12" />
                <span>PID: {{ process.pid }}</span>
              </div>
              <div class="meta-item" v-if="process.restarts > 0">
                <RefreshCw :size="12" />
                <span>{{ appStore.t('processes.restarts') }}: {{ process.restarts }}</span>
              </div>
            </div>
            <div class="card-actions">
              <Tooltip :title="appStore.t('actions.restart')">
                <Button
                  size="small"
                  :disabled="process.status === 'stopped'"
                  @click="handleRestart(process)"
                >
                  <template #icon><RotateCw :size="14" /></template>
                </Button>
              </Tooltip>
              <Tooltip :title="appStore.t('actions.logs')">
                <Button size="small" @click="handleLogs(process)">
                  <template #icon><FileText :size="14" /></template>
                </Button>
              </Tooltip>
              <Tooltip :title="appStore.t('actions.settings')">
                <Button size="small" @click="openEditModal(process)" :disabled="process.status === 'running' || process.status === 'starting'">
                  <template #icon><Pencil :size="14" /></template>
                </Button>
              </Tooltip>
              <Tooltip :title="process.status === 'running' || process.status === 'starting' ? appStore.t('actions.stop') : appStore.t('actions.start')">
                <Button
                  :type="process.status === 'running' || process.status === 'starting' ? 'default' : 'primary'"
                  :danger="process.status === 'running' || process.status === 'starting'"
                  size="small"
                  @click="handleStart(process)"
                >
                  <template #icon>
                    <Square v-if="process.status === 'running' || process.status === 'starting'" :size="14" />
                    <Play v-else :size="14" />
                  </template>
                </Button>
              </Tooltip>
            </div>
          </div>
        </Spin>
      </div>
    </div>

    <!-- 模态框 -->
    <AddProcessModal v-model:visible="showModal" />
    <EditProcessModal v-model:visible="showEditModal" :process="editingProcess" />
    <SettingsModal v-model:visible="showSettingsModal" />
    <LogsModal
      v-model:visible="showLogsModal"
      :process-id="logsProcess?.id || ''"
      :process-name="logsProcess?.name || ''"
    />
  </section>
</template>

<style scoped>
.process-list-container {
  @apply flex min-h-0 flex-1 flex-col rounded-xl border border-slate-200/60 bg-white/80 p-4 backdrop-blur-sm dark:border-slate-700/60 dark:bg-slate-800/80;
}

.process-header {
  @apply flex flex-wrap items-center justify-between gap-4 pb-4;
}

.header-left {
  @apply flex items-center gap-3;
}

.header-title {
  @apply flex items-center gap-2 text-lg font-bold text-slate-800 dark:text-slate-200;
}

.title-icon {
  @apply text-emerald-600 dark:text-emerald-400;
}

.process-count {
  @apply rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600 dark:bg-slate-700 dark:text-slate-400;
}

.header-actions {
  @apply flex items-center gap-2;
}

.search-input {
  @apply w-48;
}

.process-grid {
  @apply flex-1 overflow-y-auto;
}

.empty-state {
  @apply flex flex-col items-center justify-center py-16 text-center;
}

.empty-icon {
  @apply mb-4 text-slate-300 dark:text-slate-600;
}

.empty-text {
  @apply mb-4 text-slate-500 dark:text-slate-400;
}

.process-card {
  @apply mb-3 rounded-xl border border-slate-200/80 bg-gradient-to-br from-white to-slate-50/50 p-4 transition-all duration-300 hover:shadow-md hover:border-slate-300 dark:border-slate-700/80 dark:from-slate-800 dark:to-slate-900/50 dark:hover:border-slate-600;
}

.process-card-running {
  @apply border-l-4 border-l-emerald-500;
}

.card-header {
  @apply flex items-center justify-between mb-3;
}

.card-title-section {
  @apply flex items-center gap-2;
}

.status-indicator {
  @apply h-2.5 w-2.5 rounded-full;
}

.status-dot-running {
  @apply bg-emerald-500 shadow-lg shadow-emerald-500/50;
  animation: pulse 2s infinite;
}

.status-dot-stopped {
  @apply bg-slate-400;
}

.status-dot-failed {
  @apply bg-red-500 shadow-lg shadow-red-500/50;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.card-title {
  @apply text-base font-semibold text-slate-800 dark:text-slate-200;
}

.status-tag {
  @apply text-xs font-medium;
}

.card-command {
  @apply mb-3 rounded-lg bg-slate-100/80 px-3 py-2 dark:bg-slate-900/50;
}

.command-text {
  @apply text-xs text-slate-600 dark:text-slate-400 font-mono break-all;
}

.card-error {
  @apply mb-3 rounded-lg bg-red-50 px-3 py-2 dark:bg-red-900/20;
}

.error-label {
  @apply text-xs font-semibold text-red-600 dark:text-red-400 mr-1;
}

.error-text {
  @apply text-xs text-red-600 dark:text-red-400;
}

.card-meta {
  @apply flex flex-wrap items-end justify-between gap-4 text-xs text-slate-500 dark:text-slate-400;
}

.meta-left {
  @apply flex flex-wrap items-center gap-4;
}

.meta-item {
  @apply flex items-center gap-1;
}

.meta-label {
  @apply ml-1;
}

.card-actions {
  @apply flex items-center gap-2;
}
</style>
