<script lang="ts" setup>
import { Modal } from 'ant-design-vue'
import { useAppStore } from '../stores/app'

defineProps<{ open: boolean; message: string }>()
const emit = defineEmits<{ 'update:open': [boolean] }>()

const appStore = useAppStore()
</script>

<template>
  <Modal
    :open="open"
    :title="appStore.t('messages.quitBlockedTitle')"
    :footer="null"
    :width="'min(420px, 90vw)'"
    @cancel="emit('update:open', false)"
  >
    <p class="text-sm leading-relaxed">{{ message }}</p>
  </Modal>
</template>
