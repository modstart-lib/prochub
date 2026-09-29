<script lang="ts" setup>
import { Switch } from 'ant-design-vue';
import { Rocket } from 'lucide-vue-next';
import { computed, onMounted, onUnmounted } from 'vue';
import { SetAutoStartEnabled } from '../../../wailsjs/go/main/App';
import { useAppStore } from '../../stores/app';
import { testActionSet, testActionUnset } from '../../utils/test';
import SettingSection from './SettingSection.vue';

const appStore = useAppStore()

// 开机自启状态由 Store 持有，CLI 或托盘改动后会自动同步到这里
const autoStart = computed(() => appStore.autoStart)

const toggleAutoStart = async (checked: boolean | string | number) => {
  try {
    await SetAutoStartEnabled(!!checked)
    // 后端会广播 config:changed，界面状态由 Store 自动回填，无需本地赋值
  } catch (e) {
    console.error('Failed to update auto-start:', e)
  }
}

onMounted(() => {
  testActionSet('Setting.getAutoStart', () => autoStart.value)
  testActionSet('Setting.setAutoStart', async (params: unknown) => {
    const { enabled } = params as { enabled: boolean }
    await toggleAutoStart(enabled)
    return autoStart.value
  })
})

onUnmounted(() => {
  testActionUnset(['Setting.getAutoStart', 'Setting.setAutoStart'])
})
</script>

<template>
  <SettingSection :title="appStore.t('settings.autoStart.title')" :desc="appStore.t('settings.autoStart.desc')">
    <template #icon>
      <Rocket class="w-5 h-5" aria-hidden="true" />
    </template>
    <template #control>
      <Switch :checked="autoStart" @change="toggleAutoStart" />
    </template>
  </SettingSection>
</template>
