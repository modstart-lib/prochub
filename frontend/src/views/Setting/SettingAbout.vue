<script lang="ts" setup>
import { Button, Card, Modal } from 'ant-design-vue';
import { Globe, Info, MessageSquare } from 'lucide-vue-next';
import { onMounted, onUnmounted, ref } from 'vue';
import { GetAppConfig, GetAppName, GetPlatform, GetProcessLogs, GetSystemLogs, GetSystemVersion, ListProcesses } from '../../../wailsjs/go/main/App';
import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime';
import { getAppVersion } from '../../services/version';
import { useAppStore } from '../../stores/app';

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

const copyGithubLink = () => {
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
  <Card class="about-card" :bordered="false">
    <template #title>
      <div class="about-header">
        <Info :size="16" class="about-icon" />
        <span>{{ appStore.t('settings.about') }}</span>
      </div>
    </template>
    <div class="about-content">
      <p class="about-text">{{ appStore.t('settings.aboutDesc') }}</p>
      <div class="about-meta">
        <span
          class="github-link"
          @click="copyGithubLink"
          role="button"
          tabindex="0"
        >
          <Globe :size="14" />
          github.com/modstart-lib/prochub
        </span>
        <Button
          v-if="feedbackUrl"
          type="primary"
          size="small"
          class="feedback-btn ml-auto"
          @click="showFeedbackModal = true"
        >
          <template #icon>
            <MessageSquare :size="14" />
          </template>
          工单反馈
        </Button>
      </div>
    </div>
  </Card>

  <Modal
    v-model:open="showFeedbackModal"
    title="工单反馈"
    :footer="null"
    width="600px"
    style="top: 20px"
    :bodyStyle="{ padding: 0, height: '70vh', overflow: 'visible' }"
  >
    <div style="width:calc(48px + 100%); height:calc(20px + 100%);overflow:hidden;border-radius:0 0 8px 8px;margin:0px 24px -20px -24px;">
      <iframe
        v-if="feedbackUrl"
        :src="feedbackUrl"
        style="width: 100%; height: 100%;"
      ></iframe>
    </div>
  </Modal>
</template>

<style scoped>
.about-card {
  @apply rounded-xl bg-slate-50 dark:bg-slate-800/50;
}

.about-card :deep(.ant-card-head) {
  @apply border-b-0 pb-0;
}

.about-card :deep(.ant-card-body) {
  @apply pt-2;
}

.about-header {
  @apply flex items-center gap-2 text-sm font-semibold text-slate-700 dark:text-slate-300;
}

.about-icon {
  @apply text-slate-500;
}

.about-content {
  @apply flex flex-col gap-3;
}

.about-text {
  @apply text-sm text-slate-600 dark:text-slate-400 leading-relaxed;
}

.about-meta {
  @apply flex items-center gap-2;
}

.github-link {
  @apply flex items-center gap-1.5 text-xs text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300 transition-colors cursor-pointer;
}

.feedback-btn {
  @apply flex items-center gap-1.5 text-xs;
}
</style>
