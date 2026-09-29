<script lang="ts" setup>
import { onMounted } from 'vue';
import { trackVisit } from '../services/analytics';
import { useAppStore } from '../stores/app';
import SettingAbout from './Setting/SettingAbout.vue';
import SettingAutoStart from './Setting/SettingAutoStart.vue';
import SettingDataDir from './Setting/SettingDataDir.vue';
import SettingGroup from './Setting/SettingGroup.vue';
import SettingLanguage from './Setting/SettingLanguage.vue';
import SettingTheme from './Setting/SettingTheme.vue';
import SettingVersion from './Setting/SettingVersion.vue';
import SettingWSL from './Setting/SettingWSL.vue';

const appStore = useAppStore()

onMounted(() => {
  trackVisit('Settings')
})
</script>

<template>
  <div class="settings-page">
    <div class="settings-header">
      <div class="settings-title">{{ appStore.t('settings.title') }}</div>
    </div>

    <div class="settings-content">
      <SettingGroup :title="appStore.t('settings.group.appearance')">
        <SettingTheme />
        <SettingLanguage />
      </SettingGroup>

      <SettingGroup :title="appStore.t('settings.group.startup')">
        <SettingAutoStart />
      </SettingGroup>

      <SettingWSL />

      <SettingGroup :title="appStore.t('settings.group.general')">
        <SettingVersion />
        <SettingDataDir />
        <SettingAbout />
      </SettingGroup>
    </div>
  </div>
</template>

<style scoped>
.settings-page {
  @apply flex flex-col rounded-xl border border-slate-200/60 bg-white/80 p-4 backdrop-blur-sm dark:border-slate-700/60 dark:bg-slate-800/80;
}

.settings-header {
  @apply border-b border-slate-200 pb-4 dark:border-slate-700;
}

.settings-title {
  @apply text-xl font-bold text-slate-800 dark:text-slate-200;
}

.settings-content {
  @apply flex flex-col gap-6 pt-4;
}
</style>
