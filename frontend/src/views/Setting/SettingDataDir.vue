<script lang="ts" setup>
import { Button, Divider } from 'ant-design-vue';
import { FolderOpen } from 'lucide-vue-next';
import { onMounted, onUnmounted, ref } from 'vue';
import { GetDataDirInfo, OpenDataDir } from '../../../wailsjs/go/main/App';
import { useAppStore } from '../../stores/app';
import { testActionSet, testActionUnset } from '../../utils/test';
import SettingSection from './SettingSection.vue';

const appStore = useAppStore()
const dataDir = ref('')
// Default to the built-in location so the section stays hidden until the
// backend confirms a custom data root is in effect.
const isDefault = ref(true)

onMounted(async () => {
  try {
    const info = await GetDataDirInfo()
    dataDir.value = info?.path || ''
    isDefault.value = info?.isDefault !== false
  } catch (e) {
    console.error('Failed to load data directory:', e)
  }

  testActionSet('Setting.getDataDir', () => dataDir.value)
  testActionSet('Setting.isDataDirDefault', () => isDefault.value)
})

onUnmounted(() => {
  testActionUnset(['Setting.getDataDir', 'Setting.isDataDirDefault'])
})

const doOpenDataDir = async () => {
  try {
    await OpenDataDir()
  } catch (e) {
    console.error('Failed to open data directory:', e)
  }
}
</script>

<template>
  <template v-if="!isDefault">
    <Divider class="section-divider" />
    <SettingSection
      class="datadir-section"
      :title="appStore.t('settings.dataDir.title')"
      :desc="appStore.t('settings.dataDir.desc')"
    >
      <template #icon>
        <FolderOpen class="w-5 h-5" aria-hidden="true" />
      </template>
      <template #control>
        <span class="data-dir-path" :title="dataDir">{{ dataDir }}</span>
        <Button type="primary" @click="doOpenDataDir">
          <template #icon><FolderOpen class="w-4 h-4" aria-hidden="true" /></template>
          {{ appStore.t('settings.dataDir.open') }}
        </Button>
      </template>
    </SettingSection>
  </template>
</template>

<style scoped>
.section-divider {
  @apply my-2;
}

.data-dir-path {
  @apply max-w-[40vw] truncate text-xs text-slate-500 dark:text-slate-400;
}
</style>
