<script lang="ts" setup>
import { Button, InputSearch, message } from 'ant-design-vue'
import { Cpu, Plus } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import type { ProcessItem } from '../stores/app'
import { useAppStore } from '../stores/app'
import ProcessAddModal from '@/views/Process/ProcessAddModal.vue'
import ProcessEditModal from '@/views/Process/ProcessEditModal.vue'
import ProcessLogsModal from '@/views/Process/ProcessLogsModal.vue'
import ProcessCard from '@/views/Process/ProcessCard.vue'
import ProcessDashboardSummary from '@/views/Process/ProcessDashboardSummary.vue'
import AppEmptyState from '@/components/AppEmptyState.vue'
import { testActionSet } from '@/utils/test'

const appStore = useAppStore()
const showModal = ref(false)
const showEditModal = ref(false)
const showLogsModal = ref(false)
const editingProcess = ref<ProcessItem | null>(null)
const logsProcess = ref<ProcessItem | null>(null)
const searchQuery = ref('')
const loadingProcessId = ref<string | null>(null)

// Filter and sort process list by ID (ensure fixed order)
const filteredProcesses = computed(() => {
  let list = [...appStore.processes]
  // Sort by ID to ensure list order is fixed
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

  // ── 自动化测试 action ───────────────────────────────────────────────────
  testActionSet('Process.getCount', () => filteredProcesses.value.length)
  testActionSet('Process.getTotalCount', () => appStore.processes.length)
  testActionSet('Process.getProcesses', () => appStore.processes)
  testActionSet('Process.getSearch', () => searchQuery.value)
  testActionSet('Process.setSearch', (params: unknown) => {
    searchQuery.value = (params as { query: string }).query
  })
  testActionSet('Process.addModal.show', () => { showModal.value = true })
  testActionSet('Process.addModal.hide', () => { showModal.value = false })
  testActionSet('Process.addModal.isVisible', () => showModal.value)
  testActionSet('Process.editModal.isVisible', () => showEditModal.value)
  testActionSet('Process.logsModal.isVisible', () => showLogsModal.value)
  testActionSet('Process.editModal.show', (params: unknown) => {
    const { id } = params as { id: string }
    const found = appStore.processes.find((p) => p.id === id)
    if (!found) throw new Error(`进程不存在: ${id}`)
    editingProcess.value = found
    showEditModal.value = true
  })
  testActionSet('Process.editModal.hide', () => { showEditModal.value = false })
  testActionSet('Process.logsModal.show', (params: unknown) => {
    const { id } = params as { id: string }
    const found = appStore.processes.find((p) => p.id === id)
    if (!found) throw new Error(`进程不存在: ${id}`)
    logsProcess.value = found
    showLogsModal.value = true
  })
  testActionSet('Process.logsModal.hide', () => { showLogsModal.value = false })
  testActionSet('Process.start', async (params: unknown) => {
    const { id } = params as { id: string }
    await appStore.startProcess(id)
  })
  testActionSet('Process.stop', async (params: unknown) => {
    const { id } = params as { id: string }
    await appStore.stopProcess(id)
  })
  testActionSet('Process.restart', async (params: unknown) => {
    const { id } = params as { id: string }
    await appStore.restartProcess(id)
  })
  testActionSet('Process.setAutoStart', async (params: unknown) => {
    const { id, enabled } = params as { id: string; enabled: boolean }
    await appStore.setProcessAutoStart(id, enabled)
  })
  testActionSet('Process.remove', async (params: unknown) => {
    const { id } = params as { id: string }
    await appStore.removeProcess(id)
  })

  // ── 截图专用 action（prepare / cleanup）───────────────────────────────────
  
})

const openEditModal = (process: ProcessItem) => {
  editingProcess.value = process
  showEditModal.value = true
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

const handleAutoStartToggle = async (process: ProcessItem, enabled: boolean) => {
  loadingProcessId.value = process.id
  try {
    await appStore.setProcessAutoStart(process.id, enabled)
    message.success(appStore.t('messages.processUpdated'))
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
  <section class="process-page">
    <!-- Header -->
    <div class="process-header">
      <div class="header-left">
        <div class="header-title">
          <Cpu class="w-5 h-5 title-icon" aria-hidden="true" />
          {{ appStore.t('processes.title') }}
        </div>
        <span class="process-count">{{ appStore.t('processes.count', { count: filteredProcesses.length }) }}</span>
      </div>
      <div class="header-actions">
        <InputSearch
          v-model:value="searchQuery"
          :placeholder="appStore.t('actions.search')"
          class="search-input"
          allow-clear
        />
        <Button type="primary" @click="showModal = true">
          <template #icon><Plus class="w-4 h-4" aria-hidden="true" /></template>
          {{ appStore.t('actions.addProcess') }}
        </Button>
      </div>
    </div>

    <!-- Dashboard summary -->
    <ProcessDashboardSummary />

    <!-- Process List -->
    <div class="process-grid">
      <AppEmptyState v-if="filteredProcesses.length === 0" :description="appStore.t('processes.empty')" />

      <ProcessCard
        v-for="process in filteredProcesses"
        :key="process.id"
        :process="process"
        :loading="loadingProcessId === process.id"
        @start="handleStart"
        @restart="handleRestart"
        @logs="handleLogs"
        @edit="openEditModal"
        @auto-start-change="handleAutoStartToggle"
      />
    </div>

    <!-- 模态框 -->
    <ProcessAddModal v-model:visible="showModal" />
    <ProcessEditModal v-model:visible="showEditModal" :process="editingProcess" />
    <ProcessLogsModal
      v-model:visible="showLogsModal"
      :process-id="logsProcess?.id || ''"
      :process-name="logsProcess?.name || ''"
    />
  </section>
</template>

<style scoped>
.process-page {
  @apply flex flex-col gap-4 rounded-xl border border-slate-200/60 bg-white/80 p-4 backdrop-blur-sm dark:border-slate-700/60 dark:bg-slate-800/80;
}

.process-header {
  @apply flex flex-wrap items-center justify-between gap-4;
}

.header-left {
  @apply flex items-center gap-3;
}

.header-title {
  @apply flex items-center gap-2 text-xl font-bold text-slate-800 dark:text-slate-200;
}

.title-icon {
  @apply text-emerald-600 dark:text-emerald-400;
}

.process-count {
  @apply rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600 dark:bg-slate-700 dark:text-slate-400;
}

.header-actions {
  @apply flex items-center gap-4;
}

.search-input {
  @apply w-48;
}

.process-grid {
  @apply flex flex-col gap-3;
}
</style>
