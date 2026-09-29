<script lang="ts" setup>
import { Empty } from 'ant-design-vue'
import { Cpu } from 'lucide-vue-next'

withDefaults(defineProps<{ description?: string }>(), { description: '' })
</script>

<template>
  <Empty :description="description" class="app-empty">
    <template #image>
      <Cpu class="w-12 h-12 text-slate-300 dark:text-slate-600" aria-hidden="true" />
    </template>
  </Empty>
</template>

<style scoped>
.app-empty :deep(.ant-empty-description) {
  @apply text-slate-500 dark:text-slate-400;
}
</style>
