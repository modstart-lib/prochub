<script lang="ts" setup>
import { Switch } from 'ant-design-vue';
import { Power } from 'lucide-vue-next';
import { onMounted, ref } from 'vue';
import { GetAutoStartEnabled, SetAutoStartEnabled } from '../../../wailsjs/go/main/App';
import { useAppStore } from '../../stores/app';

const appStore = useAppStore()
const autoStart = ref(false)

onMounted(async () => {
  try {
    autoStart.value = await GetAutoStartEnabled()
  } catch (e) {
    console.error('Failed to load auto-start status:', e)
  }
})

const toggleAutoStart = async (checked: boolean | string | number) => {
  try {
    await SetAutoStartEnabled(!!checked)
    autoStart.value = !!checked
  } catch (e) {
    console.error('Failed to update auto-start:', e)
    autoStart.value = !checked
  }
}
</script>

<template>
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
      <Switch :checked="autoStart" @change="toggleAutoStart" />
    </div>
  </div>
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

.autostart-icon {
  @apply bg-green-100 text-green-600 dark:bg-green-900/50 dark:text-green-400;
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
</style>
