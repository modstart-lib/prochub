<script lang="ts" setup>
import { RadioButton, RadioGroup } from 'ant-design-vue';
import { Moon, Sun } from 'lucide-vue-next';
import { computed } from 'vue';
import { useAppStore } from '../../stores/app';

const appStore = useAppStore()

const themeMode = computed({
  get: () => (appStore.isDark ? 'dark' : 'light'),
  set: (value: 'light' | 'dark') => {
    appStore.setTheme(value === 'dark')
  },
})
</script>

<template>
  <div class="setting-section">
    <div class="section-header">
      <div class="section-icon theme-icon">
        <Sun v-if="!appStore.isDark" :size="18" />
        <Moon v-else :size="18" />
      </div>
      <div class="section-info">
        <h3 class="section-title">{{ appStore.t('settings.theme.title') }}</h3>
        <p class="section-desc">{{ appStore.t('settings.theme.desc') }}</p>
      </div>
    </div>
    <div class="section-control">
      <RadioGroup v-model:value="themeMode" button-style="solid">
        <RadioButton value="light" class="theme-button">
          <Sun :size="14" class="button-icon" />
          {{ appStore.t('settings.theme.light') }}
        </RadioButton>
        <RadioButton value="dark" class="theme-button">
          <Moon :size="14" class="button-icon" />
          {{ appStore.t('settings.theme.dark') }}
        </RadioButton>
      </RadioGroup>
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

.theme-icon {
  @apply bg-amber-100 text-amber-600 dark:bg-amber-900/50 dark:text-amber-400;
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

.theme-button {
  @apply flex items-center gap-1.5;
}

.button-icon {
  @apply -ml-0.5;
}

.section-control :deep(.ant-radio-group) {
  @apply flex;
}

.section-control :deep(.ant-radio-button-wrapper) {
  @apply flex items-center gap-1.5 border-slate-300 dark:border-slate-600;
}

.section-control :deep(.ant-radio-button-wrapper > span) {
  @apply flex items-center gap-1.5;
}

.section-control :deep(.ant-radio-button-wrapper:first-child) {
  @apply rounded-l-lg;
}

.section-control :deep(.ant-radio-button-wrapper:last-child) {
  @apply rounded-r-lg;
}

.section-control :deep(.ant-radio-button-wrapper-checked) {
  @apply bg-indigo-500 border-indigo-500 text-white;
}

.section-control :deep(.ant-radio-button-wrapper-checked:hover) {
  @apply bg-indigo-600 border-indigo-600;
}

.section-control :deep(.ant-radio-button-wrapper:not(.ant-radio-button-wrapper-checked)) {
  @apply bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300;
}

.section-control :deep(.ant-radio-button-wrapper:not(.ant-radio-button-wrapper-checked):hover) {
  @apply text-indigo-500 dark:text-indigo-400;
}
</style>
