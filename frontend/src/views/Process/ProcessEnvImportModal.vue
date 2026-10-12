<script lang="ts" setup>
import { Modal, Textarea, message } from 'ant-design-vue'
import { computed, ref, watch } from 'vue'
import { useAppStore } from '../../stores/app'
import { parseEnvText, type EnvEntry } from '../../utils/env'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{
  'update:visible': [boolean]
  import: [EnvEntry[]]
}>()

const appStore = useAppStore()
const text = ref('')
const parsedCount = computed(() => parseEnvText(text.value).length)

watch(
  () => props.visible,
  (visible) => {
    if (visible) text.value = ''
  },
)

const handleImport = () => {
  const entries = parseEnvText(text.value)
  if (!entries.length) {
    message.warning(appStore.t('processes.env.importEmpty'))
    return
  }
  emit('import', entries)
  emit('update:visible', false)
}

const handleCancel = () => {
  emit('update:visible', false)
}
</script>

<template>
  <Modal
    :open="props.visible"
    :title="appStore.t('processes.env.importTitle')"
    :ok-text="appStore.t('processes.env.import')"
    :cancel-text="appStore.t('actions.cancel')"
    :ok-button-props="{ disabled: parsedCount === 0 }"
    width="min(600px, 90vw)"
    @ok="handleImport"
    @cancel="handleCancel"
  >
    <div class="env-import">
      <p class="env-import-hint">{{ appStore.t('processes.env.importHint') }}</p>
      <Textarea
        v-model:value="text"
        :rows="10"
        :placeholder="appStore.t('processes.env.importPlaceholder')"
        class="env-import-textarea"
      />
    </div>
  </Modal>
</template>

<style scoped>
.env-import {
  @apply space-y-2;
}

.env-import-hint {
  @apply text-xs text-slate-500 dark:text-slate-400;
}

.env-import-textarea {
  @apply font-mono;
}
</style>
