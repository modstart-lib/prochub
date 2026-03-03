<script lang="ts" setup>
import { Button } from 'ant-design-vue';
import { RefreshCw } from 'lucide-vue-next';
import { onMounted, ref } from 'vue';
import { checkVersionAndPrompt, getAppVersion, isAppStoreBuild } from '../../services/version';
import { useAppStore } from '../../stores/app';

const appStore = useAppStore()
const appVersion = ref('')
const versionChecking = ref(false)

onMounted(async () => {
  appVersion.value = await getAppVersion()
})

const handleCheckVersion = async () => {
  versionChecking.value = true
  try {
    await checkVersionAndPrompt({ showLatestMessage: true, showErrorMessage: true })
  } finally {
    versionChecking.value = false
  }
}
</script>

<template>
  <template v-if="!isAppStoreBuild">
    <div class="setting-section">
      <div class="section-header">
        <div class="section-icon version-icon">
          <RefreshCw :size="18" />
        </div>
        <div class="section-info">
          <h3 class="section-title">{{ appStore.t('settings.version.title') }}</h3>
          <p class="section-desc">{{ appStore.t('settings.version.desc') }}</p>
        </div>
      </div>
      <div class="section-control">
        <div class="version-control">
          <span class="current-version">{{ appStore.t('settings.version.currentVersion') }}: {{ appVersion }}</span>
          <Button
            type="primary"
            size="small"
            :loading="versionChecking"
            @click="handleCheckVersion"
          >
            <template #icon>
              <RefreshCw :size="14" v-if="!versionChecking" />
            </template>
            {{ versionChecking ? appStore.t('settings.version.checking') : appStore.t('settings.version.checkUpdate') }}
          </Button>
        </div>
      </div>
    </div>
  </template>
</template>

<style scoped>
.setting-section {
  @apply flex flex-row items-center justify-between gap-4;
}

.section-header {
  @apply flex items-center gap-3;
}

.section-icon {
  @apply flex h-10 w-10 items-center justify-center rounded-lg;
}

.version-icon {
  @apply bg-purple-100 text-purple-600 dark:bg-purple-900/50 dark:text-purple-400;
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
  @apply flex items-center;
}

.version-control {
  @apply flex items-center gap-3;
}

.current-version {
  @apply text-xs text-slate-500 dark:text-slate-400;
}
</style>
