<script lang="ts" setup>
import { Select } from 'ant-design-vue';
import { Languages } from 'lucide-vue-next';
import { computed, onMounted, onUnmounted } from 'vue';
import { useAppStore } from '../../stores/app';
import { testActionSet, testActionUnset } from '../../utils/test';
import SettingSection from './SettingSection.vue';

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

onMounted(() => {
  testActionSet('Setting.getLocale', () => appStore.locale)
  testActionSet('Setting.setLocale', (params: unknown) => {
    const { locale: l } = params as { locale: 'zh' | 'en' }
    appStore.setLocale(l)
  })
})
onUnmounted(() => {
  testActionUnset(['Setting.getLocale', 'Setting.setLocale'])
})
</script>

<template>
  <SettingSection :title="appStore.t('settings.language')" :desc="appStore.t('settings.languageDesc')">
    <template #icon>
      <Languages class="w-5 h-5" aria-hidden="true" />
    </template>
    <template #control>
      <Select v-model:value="locale" :options="languageOptions" class="language-select" />
    </template>
  </SettingSection>
</template>

<style scoped>
.language-select {
  @apply w-40;
}
</style>
