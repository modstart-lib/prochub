<script lang="ts" setup>
import { Button, Modal } from 'ant-design-vue';
import { Github, Info, MessageSquare } from 'lucide-vue-next';
import { onMounted, onUnmounted, ref } from 'vue';
import { GetAppConfig, GetAppName, GetPlatform, GetProcessLogs, GetSystemLogs, GetSystemVersion, ListProcesses } from '../../../wailsjs/go/main/App';
import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime';
import { getAppVersion } from '../../services/version';
import { useAppStore } from '../../stores/app';
import SettingSection from './SettingSection.vue';

const appStore = useAppStore()
const feedbackUrl = ref('')
const showFeedbackModal = ref(false)

onMounted(async () => {
  try {
    const config = await GetAppConfig()
    if (config && config.feedbackUrl) {
      feedbackUrl.value = config.feedbackUrl
    }
  } catch (e) {
    console.error('Failed to load app config:', e)
  }
  window.addEventListener('message', handleFeedbackMessage)
})

onUnmounted(() => {
  window.removeEventListener('message', handleFeedbackMessage)
})

const openGithub = () => {
  BrowserOpenURL('https://github.com/modstart-lib/prochub')
}

const handleFeedbackMessage = async (event: MessageEvent) => {
  if (!event.data || !event.data.type) return

  const type = event.data.type

  if (type === 'FeedbackTicket:env') {
    try {
      const name = await GetAppName()
      const version = await getAppVersion()
      const processes = await ListProcesses()
      const platform = await GetPlatform()
      const systemVersion = await GetSystemVersion()

      const envData = {
        name,
        version,
        platform,
        systemVersion,
        processes: processes.map((p: any) => ({
          name: p.definition.name,
          status: p.status,
          restarts: p.restarts
        }))
      }

      if (event.source) {
        (event.source as Window).postMessage({
          type: 'FeedbackTicket:env',
          data: envData
        }, '*')
      }
    } catch (e) {
      console.error('Failed to collect env:', e)
    }
  } else if (type === 'FeedbackTicket:log') {
    try {
      const processes = await ListProcesses()
      let allLogs = ''

      for (const p of processes) {
        const logs = await GetProcessLogs(p.definition.id)
        if (logs && logs.length > 0) {
          allLogs += `\n=== Process: ${p.definition.name} ===\n`
          allLogs += logs.map((l: any) => `[${l.timestamp}] ${l.stream}: ${l.line}`).join('\n')
          allLogs += '\n'
        }
      }

      const systemLogs = await GetSystemLogs()
      if (systemLogs) {
        allLogs += '\n' + systemLogs
      }

      const now = new Date()
      const startTime = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString()
      const endTime = now.toISOString()

      if (event.source) {
        (event.source as Window).postMessage({
          type: 'FeedbackTicket:log',
          data: { logs: allLogs, startTime, endTime }
        }, '*')
      }
    } catch (e) {
      console.error('Failed to collect logs:', e)
    }
  }
}
</script>

<template>
  <SettingSection :title="appStore.t('settings.about')" :desc="appStore.t('settings.aboutDesc')">
    <template #icon>
      <Info class="w-5 h-5" aria-hidden="true" />
    </template>
    <template #control>
      <Button aria-label="GitHub" @click="openGithub">
        <template #icon><Github class="w-4 h-4" aria-hidden="true" /></template>
        GitHub
      </Button>
      <Button v-if="feedbackUrl" type="primary" @click="showFeedbackModal = true">
        <template #icon><MessageSquare class="w-4 h-4" aria-hidden="true" /></template>
        {{ appStore.t('settings.feedback') }}
      </Button>
    </template>
  </SettingSection>

  <Modal
    v-model:open="showFeedbackModal"
    :title="appStore.t('settings.feedback')"
    :footer="null"
    width="95vw"
    :body-style="{ padding: 0 }"
  >
    <div class="feedback-frame">
      <iframe v-if="feedbackUrl" :src="feedbackUrl" class="feedback-iframe"></iframe>
    </div>
  </Modal>
</template>

<style scoped>
.feedback-frame {
  @apply h-[70vh] overflow-hidden;
}

.feedback-iframe {
  @apply h-full w-full border-0;
}
</style>
