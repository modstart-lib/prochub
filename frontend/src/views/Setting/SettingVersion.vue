<script lang="ts" setup>
import { Button } from 'ant-design-vue';
import { RefreshCw } from 'lucide-vue-next';
import { onMounted, onUnmounted, ref } from 'vue';
import { checkVersionAndPrompt, getAppVersion, isAppStoreBuild } from '../../services/version';
import { useAppStore } from '../../stores/app';
import { testActionSet, testActionUnset } from '../../utils/test';
import SettingSection from './SettingSection.vue';

const appStore = useAppStore()
const appVersion = ref('')
const versionChecking = ref(false)

onMounted(async () => {
  appVersion.value = await getAppVersion()

  testActionSet('Setting.getVersion', () => appVersion.value)
  testActionSet('Setting.getVersionChecking', () => versionChecking.value)
  testActionSet('Setting.isAppStoreBuild', () => isAppStoreBuild)
})

onUnmounted(() => {
  testActionUnset(['Setting.getVersion', 'Setting.getVersionChecking', 'Setting.isAppStoreBuild'])
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
  <SettingSection :title="appStore.t('settings.version.title')" :desc="appStore.t('settings.version.desc')">
    <template #icon>
      <RefreshCw class="w-5 h-5" aria-hidden="true" />
    </template>
    <template #control>
      <span class="current-version">
        {{ appStore.t('settings.version.currentVersion') }}: {{ appVersion }}
      </span>
      <Button
        v-if="!isAppStoreBuild"
        type="primary"
        :loading="versionChecking"
        @click="handleCheckVersion"
      >
        <template #icon>
          <RefreshCw v-if="!versionChecking" class="w-4 h-4" aria-hidden="true" />
        </template>
        {{ versionChecking ? appStore.t('settings.version.checking') : appStore.t('settings.version.checkUpdate') }}
      </Button>
    </template>
  </SettingSection>
</template>

<style scoped>
.current-version {
  @apply text-xs text-slate-500 dark:text-slate-400;
}
</style>
