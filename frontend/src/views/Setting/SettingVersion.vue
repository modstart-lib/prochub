<script lang="ts" setup>
import { Alert, Button, Progress, Tag, message } from 'ant-design-vue';
import { RefreshCw } from 'lucide-vue-next';
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime';
import {
  bindUpdateEvents,
  checkVersionAndPrompt,
  downloadUpdate,
  getAppVersion,
  installUpdate,
  isAppStoreBuild,
  updateStore,
} from '../../services/version';
import { useAppStore } from '../../stores/app';
import { testActionSet, testActionUnset } from '../../utils/test';
import SettingSection from './SettingSection.vue';

const appStore = useAppStore()
const appVersion = ref('')
const versionChecking = ref(false)

const state = computed(() => updateStore.state)
const latestVersion = computed(() => state.value?.latestVersion || '')
const summary = computed(() => state.value?.release?.summary || '')
const errorText = computed(() => state.value?.error || '')

const isDownloading = computed(() => state.value?.state === 'downloading')
const isInstalling = computed(() => state.value?.state === 'installing' || state.value?.state === 'restarting')
const canDownload = computed(() => state.value?.state === 'available' && !!state.value?.asset)
const canManualDownload = computed(() => state.value?.state === 'available' && !state.value?.asset && !!state.value?.release?.url)
const canInstall = computed(() => state.value?.state === 'ready')
const hasUpdate = computed(() => {
  const s = state.value?.state
  return s === 'available' || s === 'downloading' || s === 'ready' || s === 'installing' || s === 'restarting'
})

onMounted(async () => {
  appVersion.value = await getAppVersion()
  bindUpdateEvents()

  testActionSet('Setting.getVersion', () => appVersion.value)
  testActionSet('Setting.getVersionChecking', () => versionChecking.value)
  testActionSet('Setting.isAppStoreBuild', () => isAppStoreBuild)
})

onUnmounted(() => {
  testActionUnset(['Setting.getVersion', 'Setting.getVersionChecking', 'Setting.isAppStoreBuild'])
})

const handleCheckVersion = async () => {
  versionChecking.value = true
  try {
    await checkVersionAndPrompt({ showLatestMessage: true, showErrorMessage: true })
  } finally {
    versionChecking.value = false
  }
}

const handleDownload = async () => {
  try {
    await downloadUpdate()
  } catch (e) {
    message.error(appStore.t('settings.version.checkFailed'))
  }
}

const handleManualDownload = () => {
  const url = state.value?.release?.url
  if (url) {
    BrowserOpenURL(url)
  }
}

const handleInstall = async () => {
  try {
    await installUpdate()
  } catch (e) {
    const text = e instanceof Error ? e.message : String(e)
    if (text.includes('stop all processes')) {
      message.error(appStore.t('settings.version.stopProcessesFirst'))
    } else {
      message.error(appStore.t('settings.version.installFailed'))
    }
  }
}
</script>

<template>
  <div class="version-panel">
    <SettingSection :title="appStore.t('settings.version.title')" :desc="appStore.t('settings.version.desc')">
      <template #icon>
        <RefreshCw class="w-5 h-5" aria-hidden="true" />
      </template>
      <template #control>
        <span class="current-version">
          {{ appStore.t('settings.version.currentVersion') }}: {{ appVersion }}
        </span>
        <Button
          v-if="!isAppStoreBuild && canDownload"
          type="primary"
          @click="handleDownload"
        >
          {{ appStore.t('settings.version.downloadUpdate') }}
        </Button>
        <Button
          v-if="!isAppStoreBuild && canManualDownload"
          type="primary"
          @click="handleManualDownload"
        >
          {{ appStore.t('settings.version.openDownloadPage') }}
        </Button>
        <Button
          v-if="!isAppStoreBuild && canInstall"
          type="primary"
          @click="handleInstall"
        >
          {{ appStore.t('settings.version.installUpdate') }}
        </Button>
        <Button
          v-if="!isAppStoreBuild"
          :loading="versionChecking"
          :disabled="isDownloading || isInstalling"
          @click="handleCheckVersion"
        >
          <template #icon>
            <RefreshCw v-if="!versionChecking" class="w-4 h-4" aria-hidden="true" />
          </template>
          {{ versionChecking ? appStore.t('settings.version.checking') : appStore.t('settings.version.checkUpdate') }}
        </Button>
      </template>
    </SettingSection>

    <div v-if="hasUpdate" class="update-detail">
      <div class="update-line">
        <Tag color="green">{{ latestVersion }}</Tag>
        <span class="update-summary">{{ summary }}</span>
      </div>
      <Progress
        v-if="isDownloading"
        :percent="state?.progress || 0"
        size="small"
        :status="'active'"
      />
      <div v-if="isDownloading" class="update-hint">
        {{ appStore.t('settings.version.downloading') }}
      </div>
      <div v-if="canInstall" class="update-hint">
        {{ appStore.t('settings.version.updateReady') }}
      </div>
      <div v-if="isInstalling" class="update-hint">
        {{ appStore.t('settings.version.installing') }}
      </div>
    </div>

    <Alert
      v-if="state?.state === 'failed' && errorText"
      class="update-error"
      type="error"
      show-icon
      :message="appStore.t('settings.version.updateFailed')"
      :description="errorText"
    />
  </div>
</template>

<style scoped>
.version-panel {
  @apply flex flex-col gap-2;
}

.current-version {
  @apply text-xs text-slate-500 dark:text-slate-400;
}

.update-detail {
  @apply flex flex-col gap-1 rounded-lg bg-slate-50 px-4 py-3 dark:bg-slate-800/50;
}

.update-line {
  @apply flex items-center gap-2;
}

.update-summary {
  @apply min-w-0 truncate text-xs text-slate-600 dark:text-slate-300;
}

.update-hint {
  @apply text-xs text-slate-500 dark:text-slate-400;
}

.update-error {
  @apply mt-1;
}
</style>
