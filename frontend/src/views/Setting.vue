<script lang="ts" setup>
import { Divider } from 'ant-design-vue';
import { onMounted } from 'vue';
import { trackVisit } from '../services/analytics';
import { isAppStoreBuild } from '../services/version';
import { useAppStore } from '../stores/app';
import SettingAbout from './Setting/SettingAbout.vue';
import SettingAutoStart from './Setting/SettingAutoStart.vue';
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
      <h2 class="settings-title">{{ appStore.t('settings.title') }}</h2>
    </div>

    <div class="settings-content">
      <SettingTheme />
      <Divider class="section-divider" />
      <SettingLanguage />
      <Divider class="section-divider" />
      <SettingAutoStart />
      <Divider v-if="!isAppStoreBuild" class="section-divider" />
      <SettingVersion />
      <Divider class="section-divider" />
      <SettingAbout />
    </div>
  </div>
</template>

<style scoped>
.settings-page {
  @apply flex flex-col rounded-xl border border-slate-200/60 bg-white/80 backdrop-blur-sm dark:border-slate-700/60 dark:bg-slate-800/80 p-6;
}

.settings-header {
  @apply pb-4 border-b border-slate-200 dark:border-slate-700;
}

.settings-title {
  @apply text-xl font-bold text-slate-800 dark:text-slate-200;
}

.settings-content {
  @apply pt-6;
}

.section-divider {
  @apply my-4;
}
</style>
