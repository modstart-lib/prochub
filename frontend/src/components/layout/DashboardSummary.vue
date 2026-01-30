<script lang="ts" setup>
import { AlertCircle, PlayCircle, StopCircle } from 'lucide-vue-next';
import { useAppStore } from '../../stores/app';

const appStore = useAppStore()
</script>

<template>
  <section class="dashboard-summary">
    <div class="grid gap-4 sm:grid-cols-3">
      <!-- 运行中 -->
      <div class="stat-card stat-card-running group">
        <div class="stat-icon-wrapper stat-icon-running">
          <PlayCircle class="stat-icon" :size="24" />
        </div>
        <div class="stat-content">
          <p class="stat-value text-emerald-600 dark:text-emerald-400">
            {{ appStore.runningCount }}
          </p>
          <p class="stat-label">
            {{ appStore.t('dashboard.running') }}
          </p>
        </div>
        <div class="stat-indicator stat-indicator-running"></div>
      </div>

      <!-- 已停止 -->
      <div class="stat-card stat-card-stopped group">
        <div class="stat-icon-wrapper stat-icon-stopped">
          <StopCircle class="stat-icon" :size="24" />
        </div>
        <div class="stat-content">
          <p class="stat-value text-slate-600 dark:text-slate-400">
            {{ appStore.stoppedCount }}
          </p>
          <p class="stat-label">
            {{ appStore.t('dashboard.stopped') }}
          </p>
        </div>
        <div class="stat-indicator stat-indicator-stopped"></div>
      </div>

      <!-- 失败 -->
      <div class="stat-card stat-card-failed group">
        <div class="stat-icon-wrapper stat-icon-failed">
          <AlertCircle class="stat-icon" :size="24" />
        </div>
        <div class="stat-content">
          <p class="stat-value text-red-600 dark:text-red-400">
            {{ appStore.failedCount }}
          </p>
          <p class="stat-label">
            {{ appStore.t('dashboard.failed') }}
          </p>
        </div>
        <div class="stat-indicator stat-indicator-failed"></div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.dashboard-summary {
  @apply rounded-xl border border-slate-200/60 bg-white/80 p-4 backdrop-blur-sm dark:border-slate-700/60 dark:bg-slate-800/80;
}

.stat-card {
  @apply relative flex items-center gap-4 rounded-xl border border-slate-200/60 bg-gradient-to-br from-white to-slate-50 p-4 transition-all duration-300 hover:shadow-lg hover:-translate-y-0.5 dark:border-slate-700/60 dark:from-slate-800 dark:to-slate-900;
  overflow: hidden;
}

.stat-icon-wrapper {
  @apply flex h-12 w-12 items-center justify-center rounded-xl transition-transform duration-300 group-hover:scale-110;
}

.stat-icon-running {
  @apply bg-emerald-100 text-emerald-600 dark:bg-emerald-900/50 dark:text-emerald-400;
}

.stat-icon-stopped {
  @apply bg-slate-100 text-slate-600 dark:bg-slate-700/50 dark:text-slate-400;
}

.stat-icon-failed {
  @apply bg-red-100 text-red-600 dark:bg-red-900/50 dark:text-red-400;
}

.stat-icon {
  @apply transition-transform duration-300;
}

.stat-content {
  @apply flex flex-col;
}

.stat-value {
  @apply text-2xl font-bold leading-none;
}

.stat-label {
  @apply mt-1 text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400;
}

.stat-indicator {
  @apply absolute right-0 top-0 h-full w-1 rounded-r-xl transition-all duration-300;
}

.stat-indicator-running {
  @apply bg-emerald-500;
}

.stat-indicator-stopped {
  @apply bg-slate-400;
}

.stat-indicator-failed {
  @apply bg-red-500;
}

.stat-card:hover .stat-indicator {
  @apply w-1.5;
}
</style>
