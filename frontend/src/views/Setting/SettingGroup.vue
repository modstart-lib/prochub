<script lang="ts" setup>
defineProps<{ title: string }>()
</script>

<template>
  <section class="setting-group">
    <div class="group-title">{{ title }}</div>
    <div class="group-body">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.setting-group {
  @apply flex flex-col gap-3;
}

.group-title {
  @apply text-xs font-semibold tracking-wide text-slate-400 dark:text-slate-500;
}

.group-body {
  @apply flex flex-col gap-4;
}
</style>
