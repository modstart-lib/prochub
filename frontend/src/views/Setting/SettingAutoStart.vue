<script lang="ts" setup>
import { Button, message, Switch } from 'ant-design-vue';
import { Power, Terminal } from 'lucide-vue-next';
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import {
  GetPlatform,
  GetWSLStatus,
  RestartWSL,
  SetAutoStartEnabled,
  SetAutoStartWSLEnabled,
  StartWSL,
  StopWSL,
} from '../../../wailsjs/go/main/App';
import { useAppStore } from '../../stores/app';
import { testActionSet, testActionUnset } from '../../utils/test';
import SettingSection from './SettingSection.vue';

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
const platform = ref('')
const wslStatus = ref<WSLStatus | null>(null)
const starting = ref(false)
const stopping = ref(false)
const restarting = ref(false)

// 开机自启 / WSL 自启状态统一由 Store 持有，CLI 或托盘改动后会自动同步到这里
const autoStart = computed(() => appStore.autoStart)
const wslEnabled = computed(() => appStore.autoStartWSL)

const isWindows = computed(() => platform.value === 'windows')

// WSL cold boots can take tens of seconds; treat both the local in-flight
// action and the backend-reported flag as "starting".
const wslStarting = computed(
  () => starting.value || restarting.value || !!wslStatus.value?.starting,
)

// WSL 的启停与「开机时自动启动 WSL」互不依赖：只要在 Windows 且 WSL 可用，
// 就允许随时查看状态并手动启停，不需要先开启应用开机自启。
const wslBusy = computed(() => starting.value || stopping.value || restarting.value)

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
  return !!status?.available && !status.running && !status.starting && !wslBusy.value
})

const canStop = computed(() => {
  const status = wslStatus.value
  return !!status?.available && status.running && !wslBusy.value
})

const canRestart = computed(() => {
  const status = wslStatus.value
  return !!status?.available && !wslBusy.value
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
    // 后端会广播 config:changed，界面状态由 Store 自动回填，无需本地赋值
  } catch (e) {
    console.error('Failed to update auto-start:', e)
  }
}

const toggleWSL = async (checked: boolean | string | number) => {
  // WSL 自启是独立设置，不依赖应用开机自启，也不影响手动启停
  try {
    await SetAutoStartWSLEnabled(!!checked)
  } catch (e) {
    console.error('Failed to update WSL auto-start:', e)
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

const handleStop = async () => {
  stopping.value = true
  try {
    await StopWSL()
  } catch (e) {
    console.error('Failed to stop WSL:', e)
    message.error(appStore.t('settings.wsl.stopFailed'))
  } finally {
    stopping.value = false
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

// Windows 下即开始轮询 WSL 状态，与开机自启设置无关
watch(isWindows, (windows) => {
  if (windows) {
    void refreshWSLStatus()
    startPolling()
  } else {
    stopPolling()
  }
}, { immediate: true })

onMounted(async () => {
  try {
    platform.value = await GetPlatform()
  } catch (e) {
    console.error('Failed to load platform:', e)
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
    <SettingSection :title="appStore.t('settings.autoStart.title')" :desc="appStore.t('settings.autoStart.desc')">
      <template #icon>
        <Power class="w-5 h-5" aria-hidden="true" />
      </template>
      <template #control>
        <Switch :checked="autoStart" @change="toggleAutoStart" />
      </template>
    </SettingSection>

    <SettingSection
      v-if="isWindows"
      class="wsl-section"
      :title="appStore.t('settings.wsl.title')"
      :desc="appStore.t('settings.wsl.desc')"
    >
      <template #icon>
        <Terminal class="w-5 h-5" aria-hidden="true" />
      </template>
      <template #control>
        <span class="wsl-status">
          <span class="status-dot" :class="statusClass"></span>
          <span>{{ statusText }}</span>
        </span>
        <Button :disabled="!canStart" :loading="starting" @click="handleStart">
          {{ appStore.t('settings.wsl.start') }}
        </Button>
        <Button :disabled="!canStop" :loading="stopping" @click="handleStop">
          {{ appStore.t('settings.wsl.stop') }}
        </Button>
        <Button :disabled="!canRestart" :loading="restarting" @click="handleRestart">
          {{ appStore.t('settings.wsl.restart') }}
        </Button>
      </template>
    </SettingSection>

    <SettingSection
      v-if="isWindows"
      class="wsl-section"
      :title="appStore.t('settings.wsl.autoStartTitle')"
      :desc="appStore.t('settings.wsl.autoStartDesc')"
    >
      <template #icon>
        <Power class="w-5 h-5" aria-hidden="true" />
      </template>
      <template #control>
        <Switch class="wsl-switch" :checked="wslEnabled" @change="toggleWSL" />
      </template>
    </SettingSection>
  </div>
</template>

<style scoped>
.autostart-group {
  @apply flex flex-col gap-4;
}

.wsl-section {
  @apply border-l-2 border-slate-200 pl-4 dark:border-slate-700;
}

.wsl-status {
  @apply flex items-center gap-1.5 rounded-md bg-slate-100 px-2 py-1 text-xs font-medium text-slate-600 dark:bg-slate-700/60 dark:text-slate-300;
}

.status-dot {
  @apply h-1.5 w-1.5 rounded-full;
}

.status-dot.running {
  @apply bg-emerald-500;
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
