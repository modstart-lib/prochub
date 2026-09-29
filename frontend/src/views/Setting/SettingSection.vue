<script lang="ts" setup>
defineProps<{ title: string; desc?: string }>()
</script>

<template>
  <div class="setting-section">
    <div class="section-header">
      <div class="section-icon">
        <slot name="icon" />
      </div>
      <div class="section-info">
        <div class="section-title">{{ title }}</div>
        <p v-if="desc" class="section-desc">{{ desc }}</p>
      </div>
    </div>
    <div class="section-control">
      <slot name="control" />
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
  @apply flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-100 text-emerald-600 dark:bg-emerald-900/50 dark:text-emerald-400;
}

.section-info {
  @apply flex min-w-0 flex-col;
}

.section-title {
  @apply text-sm font-semibold text-slate-800 dark:text-slate-200;
}

.section-desc {
  @apply text-xs text-slate-500 dark:text-slate-400;
}

.section-control {
  @apply flex shrink-0 items-center gap-2;
}
</style>
