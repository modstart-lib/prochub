<script lang="ts" setup>
import { Select } from 'ant-design-vue';
import { Globe, Languages } from 'lucide-vue-next';
import { computed } from 'vue';
import { useAppStore } from '../../stores/app';

const appStore = useAppStore()

const locale = computed({
  get: () => appStore.locale,
  set: (value: 'zh' | 'en') => {
    appStore.setLocale(value)
  },
})

const languageOptions = [
  { value: 'zh', label: '中文' },
  { value: 'en', label: 'English' },
]
</script>

<template>
  <div class="setting-section">
    <div class="section-header">
      <div class="section-icon language-icon">
        <Languages :size="18" />
      </div>
      <div class="section-info">
        <h3 class="section-title">{{ appStore.t('settings.language') }}</h3>
        <p class="section-desc">{{ appStore.t('settings.languageDesc') }}</p>
      </div>
    </div>
    <div class="section-control">
      <Select
        v-model:value="locale"
        :options="languageOptions"
        class="language-select"
        size="middle"
      >
        <template #suffixIcon>
          <Globe :size="14" />
        </template>
      </Select>
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

.language-icon {
  @apply bg-blue-100 text-blue-600 dark:bg-blue-900/50 dark:text-blue-400;
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

.language-select {
  @apply w-40;
}
</style>
