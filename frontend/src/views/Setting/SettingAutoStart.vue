<script lang="ts" setup>
import { Button, message, Switch } from 'ant-design-vue';
import { Power, Terminal } from 'lucide-vue-next';
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import {
  GetAutoStartEnabled,
  GetAutoStartWSLEnabled,
  GetPlatform,
  GetWSLStatus,
  RestartWSL,
  SetAutoStartEnabled,
  SetAutoStartWSLEnabled,
  StartWSL,
} from '../../../wailsjs/go/main/App';
import { useAppStore } from '../../stores/app';
import { testActionSet, testActionUnset } from '../../utils/test';

interface WSLDistro {
  name: string
  state: string
  version: string
  default: boolean
}

interface WSLStatus {
  supported: boolean
  available: boolean
  running: boolean
  starting: boolean
  version: string
  defaultDistro: string
  distros: WSLDistro[]
}

const appStore = useAppStore()
const autoStart = ref(false)
const platform = ref('')
const wslEnabled = ref(false)
const wslStatus = ref<WSLStatus | null>(null)
const starting = ref(false)
const restarting = ref(false)

const isWindows = computed(() => platform.value === 'windows')
// The WSL option only takes effect while the app itself auto-starts on boot.
const wslActive = computed(() => isWindows.value && autoStart.value && wslEnabled.value)

// WSL cold boots can take tens of seconds; treat both the local in-flight
// action and the backend-reported flag as "starting".
const wslStarting = computed(
  () => starting.value || restarting.value || !!wslStatus.value?.starting,
)

const statusText = computed(() => {
  const status = wslStatus.value
  // "starting" wins while WSL boots: intermediate probes may briefly report the
  // distribution as missing/stopped, which would otherwise flash "not installed".
  if (wslStarting.value) {
    return appStore.t('settings.wsl.statusStarting')
  }
  if (!status || !status.supported || !status.available) {
    return appStore.t('settings.wsl.statusUnavailable')
  }
  return status.running
    ? appStore.t('settings.wsl.statusRunning')
    : appStore.t('settings.wsl.statusStopped')
})

const statusClass = computed(() => {
  const status = wslStatus.value
  if (wslStarting.value) return 'starting'
  if (!status || !status.supported || !status.available) return 'unavailable'
  return status.running ? 'running' : 'stopped'
})

const canStart = computed(() => {
  const status = wslStatus.value
  return (
    !!status?.available && !status.running && !status.starting && !starting.value && !restarting.value
  )
})

const canRestart = computed(() => {
  const status = wslStatus.value
  return !!status?.available && !starting.value && !restarting.value
})

let pollTimer: ReturnType<typeof setInterval> | undefined
let refreshInFlight = false

const stopPolling = () => {
  if (pollTimer !== undefined) {
    clearInterval(pollTimer)
    pollTimer = undefined
  }
}

const startPolling = () => {
  stopPolling()
  pollTimer = setInterval(() => {
    void refreshWSLStatus()
  }, 5000)
}

const refreshWSLStatus = async () => {
  // A probe can take a while while WSL cold boots, so never stack requests.
  if (!isWindows.value || refreshInFlight) return
  refreshInFlight = true
  try {
    wslStatus.value = (await GetWSLStatus()) as unknown as WSLStatus
  } catch (e) {
    console.error('Failed to load WSL status:', e)
  } finally {
    refreshInFlight = false
  }
}

const toggleAutoStart = async (checked: boolean | string | number) => {
  try {
    await SetAutoStartEnabled(!!checked)
    autoStart.value = !!checked
  } catch (e) {
    console.error('Failed to update auto-start:', e)
    autoStart.value = !checked
  }
}

const toggleWSL = async (checked: boolean | string | number) => {
  if (!autoStart.value) return
  try {
    await SetAutoStartWSLEnabled(!!checked)
    wslEnabled.value = !!checked
  } catch (e) {
    console.error('Failed to update WSL auto-start:', e)
    wslEnabled.value = !checked
  }
}

const handleStart = async () => {
  starting.value = true
  try {
    await StartWSL()
  } catch (e) {
    console.error('Failed to start WSL:', e)
    message.error(appStore.t('settings.wsl.startFailed'))
  } finally {
    starting.value = false
    await refreshWSLStatus()
  }
}

const handleRestart = async () => {
  restarting.value = true
  try {
    await RestartWSL()
  } catch (e) {
    console.error('Failed to restart WSL:', e)
    message.error(appStore.t('settings.wsl.restartFailed'))
  } finally {
    restarting.value = false
    await refreshWSLStatus()
  }
}

watch(wslActive, (active) => {
  if (active) {
    void refreshWSLStatus()
    startPolling()
  } else {
    stopPolling()
  }
})

onMounted(async () => {
  try {
    autoStart.value = await GetAutoStartEnabled()
  } catch (e) {
    console.error('Failed to load auto-start status:', e)
  }

  try {
    platform.value = await GetPlatform()
  } catch (e) {
    console.error('Failed to load platform:', e)
  }

  try {
    wslEnabled.value = await GetAutoStartWSLEnabled()
  } catch (e) {
    console.error('Failed to load WSL auto-start status:', e)
  }

  testActionSet('Setting.getAutoStart', () => autoStart.value)
  testActionSet('Setting.setAutoStart', async (params: unknown) => {
    const { enabled } = params as { enabled: boolean }
    await toggleAutoStart(enabled)
    return autoStart.value
  })
  testActionSet('Setting.getAutoStartWSL', () => wslEnabled.value)
  testActionSet('Setting.setAutoStartWSL', async (params: unknown) => {
    const { enabled } = params as { enabled: boolean }
    await toggleWSL(enabled)
    return wslEnabled.value
  })
  testActionSet('Setting.getWSLStatus', () => wslStatus.value)
  testActionSet('Setting.getPlatform', () => platform.value)
})

onUnmounted(() => {
  stopPolling()
  testActionUnset([
    'Setting.getAutoStart',
    'Setting.setAutoStart',
    'Setting.getAutoStartWSL',
    'Setting.setAutoStartWSL',
    'Setting.getWSLStatus',
    'Setting.getPlatform',
  ])
})
</script>

<template>
  <div class="autostart-group">
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
          class="autostart-switch"
          :checked="autoStart"
          @change="toggleAutoStart"
        />
      </div>
    </div>

    <div
      v-if="isWindows"
      class="setting-section wsl-section"
      :class="{ 'wsl-inactive': !autoStart }"
    >
      <div class="section-header">
        <div class="section-icon wsl-icon">
          <Terminal :size="18" />
        </div>
        <div class="section-info">
          <h3 class="section-title">{{ appStore.t('settings.wsl.title') }}</h3>
          <p class="section-desc">
            {{ autoStart ? appStore.t('settings.wsl.desc') : appStore.t('settings.wsl.requiresAutoStart') }}
          </p>
        </div>
      </div>
      <div class="section-control">
        <template v-if="wslActive">
          <span class="wsl-status">
            <span class="status-dot" :class="statusClass"></span>
            <span class="status-text">{{ statusText }}</span>
          </span>
          <Button
            size="small"
            :disabled="!canStart"
            :loading="starting"
            @click="handleStart"
          >
            {{ appStore.t('settings.wsl.start') }}
          </Button>
          <Button
            size="small"
            :disabled="!canRestart"
            :loading="restarting"
            @click="handleRestart"
          >
            {{ appStore.t('settings.wsl.restart') }}
          </Button>
        </template>
        <Switch
          class="wsl-switch"
          :checked="wslEnabled"
          :disabled="!autoStart"
          @change="toggleWSL"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.autostart-group {
  @apply flex flex-col gap-4;
}

.setting-section {
  @apply flex flex-row items-center justify-between gap-4;
}

.wsl-section {
  @apply pl-4 border-l-2 border-slate-200 dark:border-slate-700 transition-opacity;
}

.wsl-inactive {
  @apply opacity-60;
}

.section-header {
  @apply flex items-center gap-3;
}

.section-icon {
  @apply flex h-10 w-10 items-center justify-center rounded-lg;
}

.autostart-icon {
  @apply bg-green-100 text-green-600 dark:bg-green-900/50 dark:text-green-400;
}

.wsl-icon {
  @apply bg-indigo-100 text-indigo-600 dark:bg-indigo-900/50 dark:text-indigo-400;
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
  @apply flex items-center gap-2;
}

.wsl-status {
  @apply flex items-center gap-1.5 rounded-md bg-slate-100 px-2 py-1 text-xs font-medium text-slate-600 dark:bg-slate-700/60 dark:text-slate-300;
}

.status-dot {
  @apply h-1.5 w-1.5 rounded-full;
}

.status-dot.running {
  @apply bg-green-500;
}

.status-dot.starting {
  @apply bg-sky-500;
  animation: wsl-pulse 1s ease-in-out infinite;
}

.status-dot.stopped {
  @apply bg-amber-500;
}

.status-dot.unavailable {
  @apply bg-slate-400;
}

@keyframes wsl-pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
}
</style>
