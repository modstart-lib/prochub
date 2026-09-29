<script lang="ts" setup>
import { Button, Spin, Switch, Tag, Tooltip } from 'ant-design-vue'
import { FileText, Hash, Pencil, Play, RefreshCw, RotateCw, Square } from 'lucide-vue-next'
import type { ProcessItem } from '@/stores/app'
import { useAppStore } from '@/stores/app'

const props = defineProps<{ process: ProcessItem; loading: boolean }>()
const emit = defineEmits<{
  start: [process: ProcessItem]
  restart: [process: ProcessItem]
  logs: [process: ProcessItem]
  edit: [process: ProcessItem]
  autoStartChange: [process: ProcessItem, enabled: boolean]
}>()

const appStore = useAppStore()

const isActive = (p: ProcessItem) => p.status === 'running' || p.status === 'starting'

const getStatusConfig = (status: string) => {
  if (status === 'running' || status === 'starting') {
    return { color: 'success', dotClass: 'status-dot-running' }
  }
  if (status === 'errored') {
    return { color: 'error', dotClass: 'status-dot-failed' }
  }
  return { color: 'default', dotClass: 'status-dot-stopped' }
}

const onAutoStartChange = (checked: boolean | string | number) => {
  emit('autoStartChange', props.process, !!checked)
}
</script>

<template>
  <Spin :spinning="loading">
    <div class="process-card" :class="{ 'process-card-running': isActive(process) }">
      <!-- Row 1: icon + title + status -->
      <div class="card-header">
        <div class="card-title-section">
          <div class="status-indicator" :class="getStatusConfig(process.status).dotClass"></div>
          <div class="card-title">{{ process.name }}</div>
        </div>
        <Tag :color="getStatusConfig(process.status).color" class="status-tag">
          {{ appStore.t(`processes.status.${process.status}`) }}
        </Tag>
      </div>

      <!-- Middle: command -->
      <div class="card-command">
        <code class="command-text">{{ process.command }}</code>
      </div>

      <div v-if="process.lastError" class="card-error">
        <span class="error-label">{{ appStore.t('logs.errorLabel') }}</span>
        <span class="error-text">{{ process.lastError }}</span>
      </div>

      <!-- Last row: id + actions -->
      <div class="card-meta">
        <div class="meta-left">
          <div class="meta-item">
            <Switch
              :checked="process.autoStart"
              size="small"
              :loading="loading"
              :aria-label="appStore.t('processes.autoStart')"
              @change="onAutoStartChange"
            />
            <span class="meta-label">{{ appStore.t('processes.autoStart') }}</span>
          </div>
          <div v-if="process.pid > 0" class="meta-item">
            <Hash class="w-3 h-3" aria-hidden="true" />
            <span>PID: {{ process.pid }}</span>
          </div>
          <div v-if="process.restarts > 0" class="meta-item">
            <RefreshCw class="w-3 h-3" aria-hidden="true" />
            <span>{{ appStore.t('processes.restarts') }}: {{ process.restarts }}</span>
          </div>
          <div class="meta-item meta-id" :title="process.id">#{{ process.id }}</div>
        </div>
        <div class="card-actions">
          <Tooltip :title="appStore.t('actions.restart')">
            <Button :disabled="process.status === 'stopped'" :aria-label="appStore.t('actions.restart')" @click="emit('restart', process)">
              <RotateCw class="w-4 h-4" aria-hidden="true" />
            </Button>
          </Tooltip>
          <Tooltip :title="appStore.t('actions.logs')">
            <Button :aria-label="appStore.t('actions.logs')" @click="emit('logs', process)">
              <FileText class="w-4 h-4" aria-hidden="true" />
            </Button>
          </Tooltip>
          <Tooltip :title="appStore.t('actions.settings')">
            <Button :disabled="isActive(process)" :aria-label="appStore.t('actions.settings')" @click="emit('edit', process)">
              <Pencil class="w-4 h-4" aria-hidden="true" />
            </Button>
          </Tooltip>
          <Tooltip :title="isActive(process) ? appStore.t('actions.stop') : appStore.t('actions.start')">
            <Button
              :type="isActive(process) ? 'default' : 'primary'"
              :danger="isActive(process)"
              :aria-label="isActive(process) ? appStore.t('actions.stop') : appStore.t('actions.start')"
              @click="emit('start', process)"
            >
              <Square v-if="isActive(process)" class="w-4 h-4" aria-hidden="true" />
              <Play v-else class="w-4 h-4" aria-hidden="true" />
            </Button>
          </Tooltip>
        </div>
      </div>
    </div>
  </Spin>
</template>

<style scoped>
.process-card {
  @apply rounded-xl border border-slate-200/80 bg-gradient-to-br from-white to-slate-50/50 p-4 transition-shadow duration-200 hover:shadow-md dark:border-slate-700/80 dark:from-slate-800 dark:to-slate-900/50;
}

.process-card-running {
  @apply border-l-4 border-l-emerald-500;
}

.card-header {
  @apply mb-3 flex items-center justify-between;
}

.card-title-section {
  @apply flex items-center gap-2;
}

.status-indicator {
  @apply h-2.5 w-2.5 rounded-full;
}

.status-dot-running {
  @apply bg-emerald-500;
  animation: pulse 2s infinite;
}

.status-dot-stopped {
  @apply bg-slate-400;
}

.status-dot-failed {
  @apply bg-red-500;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }

  50% {
    opacity: 0.5;
  }
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
  @apply break-all font-mono text-xs text-slate-600 dark:text-slate-400;
}

.card-error {
  @apply mb-3 rounded-lg bg-red-50 px-3 py-2 dark:bg-red-900/20;
}

.error-label {
  @apply mr-1 text-xs font-semibold text-red-600 dark:text-red-400;
}

.error-text {
  @apply text-xs text-red-600 dark:text-red-400;
}

.card-meta {
  @apply flex flex-wrap items-center justify-between gap-4 text-xs text-slate-500 dark:text-slate-400;
}

.meta-left {
  @apply flex min-w-0 flex-wrap items-center gap-4;
}

.meta-item {
  @apply flex items-center gap-1;
}

.meta-label {
  @apply ml-1;
}

.meta-id {
  @apply max-w-[240px] truncate font-mono;
}

.card-actions {
  @apply flex items-center gap-2;
}
</style>
