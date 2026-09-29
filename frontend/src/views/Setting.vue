<script lang="ts" setup>
import { Divider } from 'ant-design-vue';
import { onMounted } from 'vue';
import { trackVisit } from '../services/analytics';
import { isAppStoreBuild } from '../services/version';
import { useAppStore } from '../stores/app';
import SettingAbout from './Setting/SettingAbout.vue';
import SettingAutoStart from './Setting/SettingAutoStart.vue';
import SettingDataDir from './Setting/SettingDataDir.vue';
import SettingLanguage from './Setting/SettingLanguage.vue';
import SettingTheme from './Setting/SettingTheme.vue';
import SettingVersion from './Setting/SettingVersion.vue';

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
      <SettingTheme />
      <Divider class="section-divider" />
      <SettingLanguage />
      <Divider class="section-divider" />
      <SettingAutoStart />
      <Divider v-if="!isAppStoreBuild" class="section-divider" />
      <SettingVersion />
      <SettingDataDir />
      <Divider class="section-divider" />
      <SettingAbout />
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
  @apply flex flex-col pt-2;
}

.section-divider {
  @apply my-2;
}
</style>
