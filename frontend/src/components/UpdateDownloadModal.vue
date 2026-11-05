<script lang="ts" setup>
import { Alert, Button, Modal, Progress, Tag, message } from 'ant-design-vue'
import { computed, onMounted, ref } from 'vue'
import {
  bindUpdateEvents,
  hideUpdateToBackground,
  installUpdate,
  reopenUpdateModal,
  updateStore,
  updateUI,
} from '../services/version'
import { useAppStore } from '../stores/app'

const appStore = useAppStore()
const state = computed(() => updateStore.state)
const installError = ref('')

const visible = computed({
  get: () => updateUI.modalVisible,
  set: (value: boolean) => {
    updateUI.modalVisible = value
  },
})

const downloading = computed(() => state.value?.state === 'downloading')
const ready = computed(() => state.value?.state === 'ready')
const installing = computed(
  () => state.value?.state === 'installing' || state.value?.state === 'restarting',
)
const failed = computed(() => state.value?.state === 'failed' && !!state.value?.error)
const dependencyError = computed(() =>
  (state.value?.error || '').includes('missing system dependency'),
)
const errorDescription = computed(() => (state.value?.error || '').replace(/^install:\s*/, ''))

const progressStatus = computed<'active' | 'success' | 'normal'>(() => {
  if (downloading.value) return 'active'
  if (ready.value) return 'success'
  return 'normal'
})

const hintText = computed(() => {
  if (installing.value) return appStore.t('settings.version.installing')
  if (ready.value) return appStore.t('settings.version.updateReady')
  return appStore.t('settings.version.downloading')
})

const floatText = computed(() => {
  if (downloading.value) {
    return appStore.t('settings.version.backgroundDownloading', {
      progress: state.value?.progress || 0,
    })
  }
  return appStore.t('settings.version.updateReadyShort')
})

function formatBytes(value: number): string {
  if (!value || value <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit++
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`
}

function handleBackground() {
  hideUpdateToBackground()
}

function handleLater() {
  updateUI.modalVisible = false
  updateUI.minimized = true
}

function handleClose() {
  updateUI.modalVisible = false
  updateUI.minimized = false
}

async function handleInstall() {
  installError.value = ''
  try {
    await installUpdate()
  } catch (e) {
    const text = e instanceof Error ? e.message : String(e)
    installError.value = text
    if (text.includes('stop all processes')) {
      message.error(appStore.t('settings.version.stopProcessesFirst'))
    } else if (!text.includes('missing system dependency')) {
      message.error(appStore.t('settings.version.installFailed'))
    }
  }
}

onMounted(() => {
  bindUpdateEvents()
})
</script>

<template>
  <Modal
    v-model:open="visible"
    :title="appStore.t('settings.version.updateDownloadTitle')"
    :closable="false"
    :mask-closable="false"
    :footer="null"
    width="480px"
  >
    <div class="update-modal">
      <div class="version-row">
        <Tag color="green">{{ state?.latestVersion }}</Tag>
        <span class="summary">{{ state?.release?.summary }}</span>
      </div>

      <template v-if="failed">
        <Alert
          type="error"
          show-icon
          :message="
            dependencyError
              ? appStore.t('settings.version.missingDependency')
              : appStore.t('settings.version.updateFailed')
          "
          :description="errorDescription"
        />
        <div class="actions">
          <Button v-if="state?.downloadedPath" @click="handleInstall">
            {{ appStore.t('settings.version.retryInstall') }}
          </Button>
          <Button @click="handleClose">{{ appStore.t('settings.version.close') }}</Button>
        </div>
      </template>

      <template v-else>
        <Progress :percent="state?.progress || 0" :status="progressStatus" />
        <div v-if="downloading" class="meta">
          {{ formatBytes(state?.downloaded || 0) }} / {{ formatBytes(state?.total || 0) }}
        </div>
        <div class="hint">{{ hintText }}</div>

        <Alert
          v-if="installError"
          type="error"
          show-icon
          :message="appStore.t('settings.version.missingDependency')"
          :description="installError"
        />

        <div class="actions">
          <Button v-if="downloading" @click="handleBackground">
            {{ appStore.t('settings.version.downloadInBackground') }}
          </Button>
          <template v-if="ready">
            <Button @click="handleLater">{{ appStore.t('settings.version.later') }}</Button>
            <Button type="primary" :loading="installing" @click="handleInstall">
              {{ appStore.t('settings.version.restartAndUpdate') }}
            </Button>
          </template>
        </div>
      </template>
    </div>
  </Modal>

  <div
    v-if="updateUI.minimized && (downloading || ready)"
    class="update-float"
    @click="reopenUpdateModal"
  >
    <span>{{ floatText }}</span>
    <Progress
      v-if="downloading"
      type="circle"
      :percent="state?.progress || 0"
      :size="28"
      :show-info="false"
    />
  </div>
</template>

<style scoped>
.update-modal {
  @apply flex flex-col gap-3;
}

.version-row {
  @apply flex items-center gap-2;
}

.summary {
  @apply min-w-0 truncate text-xs text-slate-500 dark:text-slate-400;
}

.meta {
  @apply text-xs text-slate-500 dark:text-slate-400;
}

.hint {
  @apply text-sm text-slate-600 dark:text-slate-300;
}

.actions {
  @apply flex justify-end gap-2 pt-1;
}

.update-float {
  @apply fixed bottom-5 right-5 z-50 flex cursor-pointer items-center gap-2 rounded-full bg-emerald-600 px-4 py-2 text-xs font-medium text-white shadow-lg transition hover:bg-emerald-700;
}
</style>
