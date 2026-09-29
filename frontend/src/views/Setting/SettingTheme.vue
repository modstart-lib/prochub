<script lang="ts" setup>
import { RadioButton, RadioGroup } from 'ant-design-vue';
import { Moon, Sun } from 'lucide-vue-next';
import { computed, onMounted, onUnmounted } from 'vue';
import { useAppStore } from '../../stores/app';
import { testActionSet, testActionUnset } from '../../utils/test';
import SettingSection from './SettingSection.vue';

const appStore = useAppStore()

const themeMode = computed({
  get: () => (appStore.isDark ? 'dark' : 'light'),
  set: (value: 'light' | 'dark') => {
    appStore.setTheme(value === 'dark')
  },
})

onMounted(() => {
  testActionSet('Setting.getTheme', () => (appStore.isDark ? 'dark' : 'light'))
  testActionSet('Setting.setTheme', (params: unknown) => {
    const { theme } = params as { theme: 'light' | 'dark' }
    appStore.setTheme(theme === 'dark')
  })
})
onUnmounted(() => {
  testActionUnset(['Setting.getTheme', 'Setting.setTheme'])
})
</script>

<template>
  <SettingSection :title="appStore.t('settings.theme.title')" :desc="appStore.t('settings.theme.desc')">
    <template #icon>
      <Sun v-if="!appStore.isDark" class="w-5 h-5" aria-hidden="true" />
      <Moon v-else class="w-5 h-5" aria-hidden="true" />
    </template>
    <template #control>
      <RadioGroup v-model:value="themeMode" button-style="solid">
        <RadioButton value="light">
          <span class="theme-option">
            <Sun class="w-4 h-4" aria-hidden="true" />
            {{ appStore.t('settings.theme.light') }}
          </span>
        </RadioButton>
        <RadioButton value="dark">
          <span class="theme-option">
            <Moon class="w-4 h-4" aria-hidden="true" />
            {{ appStore.t('settings.theme.dark') }}
          </span>
        </RadioButton>
      </RadioGroup>
    </template>
  </SettingSection>
</template>

<style scoped>
.theme-option {
  @apply flex items-center gap-1.5;
}
</style>
