<script lang="ts" setup>
import { Button, message, Switch } from 'ant-design-vue';
import { Power, Terminal } from 'lucide-vue-next';
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import {
  GetPlatform,
  GetWSLStatus,
  RestartWSL,
  SetAutoStartWSLEnabled,
  StartWSL,
  StopWSL,
} from '../../../wailsjs/go/main/App';
import { useAppStore } from '../../stores/app';
import { testActionSet, testActionUnset } from '../../utils/test';
import SettingGroup from './SettingGroup.vue';
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

// WSL 自启状态由 Store 持有，CLI 或托盘改动后会自动同步到这里
const wslEnabled = computed(() => appStore.autoStartWSL)
const isWindows = computed(() => platform.value === 'windows')
const wslAvailable = computed(() => !!wslStatus.value?.available)
const wslRunning = computed(() => !!wslStatus.value?.running)

// WSL cold boots can take tens of seconds; treat both the local in-flight
// action and the backend-reported flag as "starting".
const wslStarting = computed(
  () => starting.value || restarting.value || !!wslStatus.value?.starting,
)

// WSL 的启停与「WSL 开机自启」互不依赖：只要在 Windows 且 WSL 可用，
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
    'Setting.getAutoStartWSL',
    'Setting.setAutoStartWSL',
    'Setting.getWSLStatus',
    'Setting.getPlatform',
  ])
})
</script>

<template>
  <SettingGroup v-if="isWindows" :title="appStore.t('settings.group.wsl')">
    <SettingSection
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
        <template v-if="wslAvailable">
          <Button v-if="wslRunning" :disabled="!canStop" :loading="stopping" @click="handleStop">
            {{ appStore.t('settings.wsl.stop') }}
          </Button>
          <Button v-else :disabled="!canStart" :loading="starting" @click="handleStart">
            {{ appStore.t('settings.wsl.start') }}
          </Button>
          <Button :disabled="!canRestart" :loading="restarting" @click="handleRestart">
            {{ appStore.t('settings.wsl.restart') }}
          </Button>
        </template>
      </template>
    </SettingSection>

    <SettingSection
      :title="appStore.t('settings.wsl.autoStartTitle')"
      :desc="appStore.t('settings.wsl.autoStartDesc')"
    >
      <template #icon>
        <Power class="w-5 h-5" aria-hidden="true" />
      </template>
      <template #control>
        <Switch :checked="wslEnabled" @change="toggleWSL" />
      </template>
    </SettingSection>
  </SettingGroup>
</template>

<style scoped>
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
