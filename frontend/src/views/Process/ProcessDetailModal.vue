<script lang="ts" setup>
import { Button, Descriptions, DescriptionsItem, Modal, Tag } from 'ant-design-vue'
import { computed, onUnmounted } from 'vue'
import { useAppStore, type ProcessItem } from '../../stores/app'
import { testActionSet, testActionUnset } from '../../utils/test'

const props = defineProps<{ visible: boolean; processId: string }>()
const emit = defineEmits<{ 'update:visible': [boolean] }>()

const appStore = useAppStore()

// Read the live item from the store so an open modal keeps following status
// changes pushed by the periodic refresh.
const detail = computed<ProcessItem | null>(
  () => appStore.processes.find((p) => p.id === props.processId) || null
)

const isActive = computed(
  () => detail.value?.status === 'running' || detail.value?.status === 'starting'
)

const statusColor = computed(() => {
  if (isActive.value) return 'success'
  return detail.value?.status === 'errored' ? 'error' : 'default'
})

const argsText = computed(() => (detail.value?.args?.length ? detail.value.args.join(' ') : '—'))
const workingDirText = computed(() => detail.value?.workingDir || '—')
const envEntries = computed(() => Object.entries(detail.value?.env ?? {}))

// Stored policies are always/on_failure/never; an empty value means the
// default (restart on failure), matching the backend behaviour.
const policyKey = (policy?: string) =>
  policy === 'always' ? 'always' : policy === 'never' ? 'never' : 'onFailure'

const restartPolicyText = computed(() =>
  appStore.t(`processes.restartPolicies.${policyKey(detail.value?.restartPolicy)}`)
)

const autoStartText = computed(() =>
  detail.value?.autoStart ? appStore.t('actions.enabled') : appStore.t('actions.disabled')
)

const closeModal = () => emit('update:visible', false)

// ── 自动化测试 action ──────────────────────────────────────────────────────
testActionSet('ProcessDetail.isVisible', () => props.visible)
testActionSet('ProcessDetail.getDetail', () => {
  const item = detail.value
  if (!item) return null
  return {
    id: item.id,
    name: item.name,
    command: item.command,
    args: item.args,
    workingDir: item.workingDir,
    env: item.env,
    autoStart: item.autoStart,
    restartPolicy: item.restartPolicy,
    maxRetries: item.maxRetries,
    status: item.status,
    pid: item.pid,
    restarts: item.restarts,
    lastError: item.lastError,
  }
})
testActionSet('ProcessDetail.close', closeModal)

onUnmounted(() => {
  testActionUnset(['ProcessDetail.isVisible', 'ProcessDetail.getDetail', 'ProcessDetail.close'])
})
</script>

<template>
  <Modal
    :open="props.visible"
    :title="appStore.t('processes.detail.title')"
    width="min(600px, 90vw)"
    @cancel="closeModal"
  >
    <div v-if="!detail" class="detail-empty">{{ appStore.t('processes.detail.notFound') }}</div>

    <div v-else class="detail-body">
      <!-- Basic configuration -->
      <div class="detail-section">
        <div class="section-title">{{ appStore.t('processes.detail.basic') }}</div>
        <Descriptions :column="1" size="small" bordered>
          <DescriptionsItem :label="appStore.t('processes.fields.name')">
            {{ detail.name || appStore.t('processes.unnamed') }}
          </DescriptionsItem>
          <DescriptionsItem :label="appStore.t('processes.fields.command')">
            <code class="detail-code">{{ detail.command }}</code>
          </DescriptionsItem>
          <DescriptionsItem :label="appStore.t('processes.fields.args')">{{ argsText }}</DescriptionsItem>
          <DescriptionsItem :label="appStore.t('processes.fields.workingDir')">{{ workingDirText }}</DescriptionsItem>
          <DescriptionsItem :label="appStore.t('processes.detail.id')">
            <span class="detail-code">{{ detail.id }}</span>
          </DescriptionsItem>
        </Descriptions>
      </div>

      <!-- Runtime state -->
      <div class="detail-section">
        <div class="section-title">{{ appStore.t('processes.detail.runtime') }}</div>
        <Descriptions :column="1" size="small" bordered>
          <DescriptionsItem :label="appStore.t('processes.detail.status')">
            <Tag :color="statusColor">{{ appStore.t(`processes.status.${detail.status}`) }}</Tag>
          </DescriptionsItem>
          <DescriptionsItem :label="appStore.t('processes.detail.pid')">
            {{ detail.pid > 0 ? detail.pid : '—' }}
          </DescriptionsItem>
          <DescriptionsItem :label="appStore.t('processes.restarts')">{{ detail.restarts }}</DescriptionsItem>
          <DescriptionsItem v-if="detail.lastError" :label="appStore.t('processes.detail.lastError')">
            <span class="detail-error">{{ detail.lastError }}</span>
          </DescriptionsItem>
        </Descriptions>
      </div>

      <!-- Advanced settings -->
      <div class="detail-section">
        <div class="section-title">{{ appStore.t('processes.detail.advanced') }}</div>
        <Descriptions :column="1" size="small" bordered>
          <DescriptionsItem :label="appStore.t('processes.fields.autoStart')">{{ autoStartText }}</DescriptionsItem>
          <DescriptionsItem :label="appStore.t('processes.fields.restartPolicy')">{{ restartPolicyText }}</DescriptionsItem>
          <DescriptionsItem :label="appStore.t('processes.fields.maxRetries')">{{ detail.maxRetries }}</DescriptionsItem>
        </Descriptions>
      </div>

      <!-- Environment variables -->
      <div class="detail-section">
        <div class="section-title">{{ appStore.t('processes.detail.env') }}</div>
        <div v-if="envEntries.length === 0" class="env-empty">{{ appStore.t('processes.detail.envEmpty') }}</div>
        <div v-else class="env-list">
          <div v-for="([key, value]) in envEntries" :key="key" class="env-row">
            <span class="env-key">{{ key }}</span>
            <span class="env-separator">=</span>
            <span class="env-value">{{ value }}</span>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <Button @click="closeModal">{{ appStore.t('actions.close') }}</Button>
    </template>
  </Modal>
</template>

<style scoped>
.detail-body {
  @apply flex flex-col gap-4;
}

.detail-section {
  @apply flex flex-col gap-2;
}

.section-title {
  @apply text-sm font-semibold text-slate-700 dark:text-slate-300;
}

.detail-code {
  @apply break-all font-mono text-xs text-slate-600 dark:text-slate-400;
}

.detail-error {
  @apply break-all text-red-600 dark:text-red-400;
}

.detail-empty {
  @apply py-6 text-center text-sm text-slate-500 dark:text-slate-400;
}

.env-list {
  @apply flex flex-col gap-1 rounded-lg border border-slate-200 bg-slate-50/60 p-2 dark:border-slate-700 dark:bg-slate-900/40;
}

.env-row {
  @apply flex items-start gap-1 text-xs;
}

.env-key {
  @apply shrink-0 font-mono font-medium text-slate-700 dark:text-slate-300;
}

.env-separator {
  @apply font-mono text-slate-400;
}

.env-value {
  @apply break-all font-mono text-slate-600 dark:text-slate-400;
}

.env-empty {
  @apply text-xs text-slate-400 dark:text-slate-500;
}
</style>
