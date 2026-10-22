<script lang="ts" setup>
import { Button, InputSearch, Segmented, Switch, message } from 'ant-design-vue'
import { Cpu, Plus } from 'lucide-vue-next'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import type { ProcessItem, ProcessStatus } from '../stores/app'
import { useAppStore } from '../stores/app'
import ProcessAddModal from '@/views/Process/ProcessAddModal.vue'
import ProcessEditModal from '@/views/Process/ProcessEditModal.vue'
import ProcessLogsModal from '@/views/Process/ProcessLogsModal.vue'
import ProcessDetailModal from '@/views/Process/ProcessDetailModal.vue'
import ProcessCard from '@/views/Process/ProcessCard.vue'
import ProcessDashboardSummary from '@/views/Process/ProcessDashboardSummary.vue'
import AppEmptyState from '@/components/AppEmptyState.vue'
import { testActionSet, testActionUnset } from '@/utils/test'

type StatusFilter = 'all' | 'running' | 'stopped' | 'errored'

const GROUP_BY_STATUS_KEY = 'processGrouped'

const appStore = useAppStore()
const showModal = ref(false)
const showEditModal = ref(false)
const showLogsModal = ref(false)
const showDetailModal = ref(false)
const editingProcess = ref<ProcessItem | null>(null)
const logsProcess = ref<ProcessItem | null>(null)
const detailProcessId = ref('')
const searchQuery = ref('')
const statusFilter = ref<StatusFilter>('all')
const loadingProcessId = ref<string | null>(null)
// Status grouping is a preference: grouped view keeps active processes on top,
// flat view keeps every card in a fixed slot so nothing jumps on start/stop.
const groupByStatus = ref(localStorage.getItem(GROUP_BY_STATUS_KEY) !== '0')

// Active processes cover both running and the transient starting state.
const isActiveStatus = (status: ProcessStatus) => status === 'running' || status === 'starting'

// Filter tabs shown in the toolbar, each with its own live count.
const statusFilterDefs: Array<{ value: StatusFilter; match: (status: ProcessStatus) => boolean }> = [
  { value: 'all', match: () => true },
  { value: 'running', match: isActiveStatus },
  { value: 'stopped', match: (status) => status === 'stopped' },
  { value: 'errored', match: (status) => status === 'errored' },
]

// Group order keeps healthy processes first and failures easy to spot.
const statusGroupDefs: Array<{ key: string; labelKey: string; match: (status: ProcessStatus) => boolean }> = [
  { key: 'running', labelKey: 'processes.status.running', match: isActiveStatus },
  { key: 'errored', labelKey: 'processes.status.errored', match: (status) => status === 'errored' },
  { key: 'stopped', labelKey: 'processes.status.stopped', match: (status) => status === 'stopped' },
]

// Filter and sort process list by ID (ensure fixed order)
const filteredProcesses = computed(() => {
  const matcher = statusFilterDefs.find((f) => f.value === statusFilter.value)!.match
  const list = appStore.processes.filter((p) => matcher(p.status))
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

// Filter options carry counts so the tab bar doubles as a status overview.
const filterOptions = computed(() =>
  statusFilterDefs.map((f) => {
    const count = appStore.processes.filter((p) => f.match(p.status)).length
    const label = f.value === 'all' ? appStore.t('processes.filter.all') : appStore.t(`processes.status.${f.value}`)
    return { value: f.value, label: `${label} ${count}` }
  })
)

// Grouped view for the "all" filter; a single status filter renders one grid.
const groupedProcesses = computed(() =>
  statusGroupDefs
    .map((g) => ({
      key: g.key,
      label: appStore.t(g.labelKey),
      items: filteredProcesses.value.filter((p) => g.match(p.status)),
    }))
    .filter((g) => g.items.length > 0)
)

const showGroupHeaders = computed(() => groupByStatus.value && statusFilter.value === 'all')

// Rendered sections: status groups when grouping is on, otherwise one flat section
// whose cards keep their ID order (positions stay put when one of them starts).
const displayGroups = computed(() =>
  groupByStatus.value
    ? groupedProcesses.value
    : [{ key: 'all', label: '', items: filteredProcesses.value }]
)

const onGroupByStatusChange = (checked: boolean | string | number) => {
  groupByStatus.value = !!checked
  localStorage.setItem(GROUP_BY_STATUS_KEY, groupByStatus.value ? '1' : '0')
}

const emptyDescription = computed(() =>
  appStore.processes.length === 0 ? appStore.t('processes.empty') : appStore.t('processes.noMatch')
)

const onStatusFilterChange = (value: string | number) => {
  statusFilter.value = value as StatusFilter
}

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
  testActionSet('Process.getStatusFilter', () => statusFilter.value)
  testActionSet('Process.setStatusFilter', (params: unknown) => {
    statusFilter.value = (params as { filter: StatusFilter }).filter
  })
  testActionSet('Process.getGroups', () =>
    displayGroups.value.map((g) => ({ key: g.key, count: g.items.length }))
  )
  testActionSet('Process.getGroupByStatus', () => groupByStatus.value)
  testActionSet('Process.setGroupByStatus', (params: unknown) => {
    onGroupByStatusChange((params as { enabled: boolean }).enabled)
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
  testActionSet('Process.detailModal.isVisible', () => showDetailModal.value)
  testActionSet('Process.detailModal.show', (params: unknown) => {
    const { id } = params as { id: string }
    const found = appStore.processes.find((p) => p.id === id)
    if (!found) throw new Error(`进程不存在: ${id}`)
    detailProcessId.value = found.id
    showDetailModal.value = true
  })
  testActionSet('Process.detailModal.hide', () => { showDetailModal.value = false })
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

onUnmounted(() => {
  testActionUnset(['Process.getStatusFilter', 'Process.setStatusFilter', 'Process.getGroups', 'Process.getGroupByStatus', 'Process.setGroupByStatus', 'Process.detailModal.isVisible', 'Process.detailModal.show', 'Process.detailModal.hide'])
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

const handleDetail = (process: ProcessItem) => {
  detailProcessId.value = process.id
  showDetailModal.value = true
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

    <!-- Status filter + grouping switch -->
    <div class="process-toolbar">
      <Segmented :value="statusFilter" :options="filterOptions" @change="onStatusFilterChange" />
      <div class="group-toggle">
        <Switch
          :checked="groupByStatus"
          size="small"
          :aria-label="appStore.t('processes.groupByStatus')"
          @change="onGroupByStatusChange"
        />
        <span class="group-toggle-label">{{ appStore.t('processes.groupByStatus') }}</span>
      </div>
    </div>

    <!-- Process List -->
    <AppEmptyState v-if="filteredProcesses.length === 0" :description="emptyDescription" />

    <TransitionGroup v-else name="process-group" tag="div" class="process-groups">
      <section v-for="group in displayGroups" :key="group.key" class="process-group">
        <div v-if="showGroupHeaders" class="group-header">
          <span class="group-dot" :class="`group-dot-${group.key}`" aria-hidden="true"></span>
          <span class="group-title">{{ group.label }}</span>
          <span class="group-count">{{ group.items.length }}</span>
        </div>
        <TransitionGroup name="process-card" tag="div" class="process-grid">
          <ProcessCard
            v-for="process in group.items"
            :key="process.id"
            :process="process"
            :loading="loadingProcessId === process.id"
            @start="handleStart"
            @restart="handleRestart"
            @logs="handleLogs"
            @detail="handleDetail"
            @edit="openEditModal"
            @auto-start-change="handleAutoStartToggle"
          />
        </TransitionGroup>
      </section>
    </TransitionGroup>

    <!-- 模态框 -->
    <ProcessAddModal v-model:visible="showModal" />
    <ProcessEditModal v-model:visible="showEditModal" :process="editingProcess" />
    <ProcessLogsModal
      v-model:visible="showLogsModal"
      :process-id="logsProcess?.id || ''"
      :process-name="logsProcess?.name || ''"
    />
    <ProcessDetailModal v-model:visible="showDetailModal" :process-id="detailProcessId" />
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

.process-toolbar {
  @apply flex flex-wrap items-center justify-between gap-3;
}

.group-toggle {
  @apply flex items-center gap-2;
}

.group-toggle-label {
  @apply text-xs text-slate-500 dark:text-slate-400;
}

.process-groups {
  @apply flex flex-col gap-4;
}

.process-group {
  @apply flex flex-col gap-3;
}

.group-header {
  @apply flex items-center gap-2;
}

.group-dot {
  @apply h-2 w-2 rounded-full;
}

.group-dot-running {
  @apply bg-emerald-500;
}

.group-dot-errored {
  @apply bg-red-500;
}

.group-dot-stopped {
  @apply bg-slate-400;
}

.group-title {
  @apply text-sm font-semibold text-slate-700 dark:text-slate-300;
}

.group-count {
  @apply rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-500 dark:bg-slate-700 dark:text-slate-400;
}

/* Responsive multi-column grid so many processes stay visible without a long scroll. */
.process-grid {
  @apply relative grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3;
}

/* Stretch cards to the row height so grid siblings line up. */
.process-grid :deep(.ant-spin-nested-loading),
.process-grid :deep(.ant-spin-container) {
  height: 100%;
}

/* Sections glide down/up when a status group appears or disappears. */
.process-group-move {
  transition: transform 0.28s cubic-bezier(0.22, 1, 0.36, 1);
}

/* Cards glide to their new slot when the grid reorders (start/stop, filter, grouping switch). */
.process-card-move {
  transition: transform 0.28s cubic-bezier(0.22, 1, 0.36, 1);
}

.process-card-enter-active {
  transition: opacity 0.24s ease, transform 0.24s ease;
}

.process-card-enter-from {
  opacity: 0;
  transform: scale(0.96) translateY(6px);
}

/* Leaving cards drop out of the flow at once, so remaining cards can glide into place. */
.process-card-leave-active {
  @apply absolute invisible;
}
</style>
