<script lang="ts" setup>
import { Modal, Radio, RadioGroup, Tag } from 'ant-design-vue'
import { computed, ref, watch } from 'vue'
import type { ProcessImportCandidate, ProcessImportMode, ProcessImportPreview } from '../../stores/app'
import { useAppStore } from '../../stores/app'

const props = defineProps<{
  visible: boolean
  preview: ProcessImportPreview | null
  importing: boolean
}>()

const emit = defineEmits<{
  'update:visible': [boolean]
  confirm: [mode: ProcessImportMode]
}>()

const appStore = useAppStore()
const mode = ref<ProcessImportMode>('skip')

// Every time the dialog opens the safe choice (keep existing processes) is
// selected again, so an earlier overwrite decision cannot leak into a new file.
watch(
  () => props.visible,
  (visible) => {
    if (visible) mode.value = 'skip'
  },
)

const fileName = computed(() => {
  const path = props.preview?.path || ''
  return path.split(/[\\/]/).pop() || path
})

const exportedAt = computed(() => {
  const raw = props.preview?.exportedAt
  if (!raw) return ''
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString(appStore.locale === 'en' ? 'en-US' : 'zh-CN')
})

const invalidCount = computed(() => {
  const preview = props.preview
  if (!preview) return 0
  return preview.total - preview.importable
})

const commandText = (item: ProcessImportCandidate) => {
  const args = item.definition.args || []
  return [item.definition.command, ...args].filter(Boolean).join(' ')
}

// Validation codes come from the backend so the dialog stays localized.
const errorText = (code: string) => {
  if (code === 'name_required') return appStore.t('processes.transfer.errorNameRequired')
  if (code === 'command_required') return appStore.t('processes.transfer.errorCommandRequired')
  return appStore.t('processes.transfer.invalidBadge')
}

const closeModal = () => emit('update:visible', false)

const handleConfirm = () => {
  if (!props.preview || props.preview.importable === 0) return
  emit('confirm', mode.value)
}
</script>

<template>
  <Modal
    :open="props.visible"
    :title="appStore.t('processes.transfer.importTitle')"
    :ok-text="appStore.t('processes.transfer.importConfirm')"
    :cancel-text="appStore.t('actions.cancel')"
    :confirm-loading="props.importing"
    :ok-button-props="{ disabled: !props.preview || props.preview.importable === 0 }"
    width="min(640px, 92vw)"
    @ok="handleConfirm"
    @cancel="closeModal"
  >
    <div v-if="props.preview" class="import-body">
      <!-- Source file -->
      <div class="import-file">
        <code class="file-name" :title="props.preview.path">{{ fileName }}</code>
        <span v-if="props.preview.appVersion" class="file-meta">{{ props.preview.appVersion }}</span>
        <span v-if="exportedAt" class="file-meta">{{ appStore.t('processes.transfer.exportedAt', { time: exportedAt }) }}</span>
      </div>

      <!-- Counts -->
      <div class="import-summary">
        <span class="summary-text">{{ appStore.t('processes.transfer.importSummary', { count: props.preview.importable }) }}</span>
        <span v-if="props.preview.conflicts" class="summary-warn">
          {{ appStore.t('processes.transfer.importConflictCount', { count: props.preview.conflicts }) }}
        </span>
        <span v-if="invalidCount" class="summary-error">
          {{ appStore.t('processes.transfer.invalidCount', { count: invalidCount }) }}
        </span>
      </div>

      <!-- Entries -->
      <div class="import-list">
        <div
          v-for="(item, index) in props.preview.processes"
          :key="`${item.definition.name}-${index}`"
          class="import-row"
          :class="{ 'import-row-invalid': item.error }"
        >
          <div class="row-head">
            <span class="row-name">{{ item.definition.name || item.definition.command || '?' }}</span>
            <Tag v-if="item.error" color="error" class="row-tag">{{ appStore.t('processes.transfer.invalidBadge') }}</Tag>
            <Tag v-else-if="item.conflict" color="warning" class="row-tag">{{ appStore.t('processes.transfer.conflictBadge') }}</Tag>
          </div>
          <div class="row-sub">
            <code v-if="!item.error" class="row-command">{{ commandText(item) }}</code>
            <span v-else class="row-error">{{ errorText(item.error) }}</span>
          </div>
        </div>
      </div>

      <!-- Same-name handling -->
      <div v-if="props.preview.conflicts > 0" class="conflict-mode">
        <div class="mode-title">
          {{ appStore.t('processes.transfer.conflictTitle', { count: props.preview.conflicts }) }}
        </div>
        <RadioGroup v-model:value="mode">
          <Radio value="skip">{{ appStore.t('processes.transfer.conflictSkip') }}</Radio>
          <Radio value="overwrite">{{ appStore.t('processes.transfer.conflictOverwrite') }}</Radio>
        </RadioGroup>
        <div v-if="mode === 'overwrite'" class="overwrite-hint">
          {{ appStore.t('processes.transfer.overwriteHint') }}
        </div>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.import-body {
  @apply flex flex-col gap-3;
}

.import-file {
  @apply flex flex-wrap items-center gap-2 rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/40;
}

.file-name {
  @apply text-sm font-medium text-slate-700 dark:text-slate-200;
}

.file-meta {
  @apply text-xs text-slate-500 dark:text-slate-400;
}

.import-summary {
  @apply flex flex-wrap items-center gap-x-3 gap-y-1 text-xs;
}

.summary-text {
  @apply text-slate-600 dark:text-slate-300;
}

.summary-warn {
  @apply text-amber-600 dark:text-amber-400;
}

.summary-error {
  @apply text-red-600 dark:text-red-400;
}

.import-list {
  @apply max-h-72 overflow-y-auto rounded-lg border border-slate-200 dark:border-slate-700;
}

.import-row {
  @apply border-b border-slate-100 px-3 py-2 last:border-b-0 dark:border-slate-700/60;
}

.import-row-invalid {
  @apply bg-red-50/50 dark:bg-red-900/10;
}

.row-head {
  @apply flex items-center gap-2;
}

.row-name {
  @apply truncate text-sm font-medium text-slate-800 dark:text-slate-200;
}

.row-tag {
  @apply m-0;
}

.row-sub {
  @apply mt-0.5;
}

.row-command {
  @apply block truncate font-mono text-xs text-slate-500 dark:text-slate-400;
}

.row-error {
  @apply text-xs text-red-600 dark:text-red-400;
}

.conflict-mode {
  @apply flex flex-col gap-2 rounded-lg border border-amber-200 bg-amber-50/60 px-3 py-2 dark:border-amber-700/60 dark:bg-amber-900/10;
}

.mode-title {
  @apply text-xs font-medium text-amber-700 dark:text-amber-400;
}

.overwrite-hint {
  @apply text-xs text-amber-700/80 dark:text-amber-400/80;
}
</style>
