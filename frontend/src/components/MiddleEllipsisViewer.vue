<script lang="ts" setup>
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

const props = withDefaults(defineProps<{ text: string; ellipsis?: string }>(), {
  ellipsis: '…',
})

const rootRef = ref<HTMLElement | null>(null)
const display = ref(props.text)
let observer: ResizeObserver | null = null
let canvas: HTMLCanvasElement | null = null

// Measure the rendered width of a string with the element's resolved font.
const measure = (text: string): number => {
  const el = rootRef.value
  if (!el) return 0
  if (!canvas) canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d')
  if (!ctx) return 0
  const cs = getComputedStyle(el)
  ctx.font = `${cs.fontStyle} ${cs.fontWeight} ${cs.fontSize} ${cs.fontFamily}`
  return ctx.measureText(text).width
}

// Keep the head and tail of the text, collapse the middle into the ellipsis.
const update = () => {
  const el = rootRef.value
  if (!el) return
  const full = props.text ?? ''
  const available = el.clientWidth
  // Hidden (width 0) or already fitting: render as-is.
  if (available <= 0 || measure(full) <= available) {
    display.value = full
    return
  }
  if (measure(props.ellipsis) > available) {
    display.value = ''
    return
  }
  // Binary search the largest head+tail character count that still fits.
  let low = 0
  let high = full.length
  let best = ''
  while (low <= high) {
    const mid = (low + high) >> 1
    const headLen = Math.ceil(mid / 2)
    const tailLen = mid - headLen
    const tail = tailLen > 0 ? full.slice(full.length - tailLen) : ''
    const candidate = full.slice(0, headLen) + props.ellipsis + tail
    if (measure(candidate) <= available) {
      best = candidate
      low = mid + 1
    } else {
      high = mid - 1
    }
  }
  display.value = best || props.ellipsis
}

const refresh = () => {
  nextTick(update)
}

onMounted(() => {
  refresh()
  observer = new ResizeObserver(refresh)
  if (rootRef.value) observer.observe(rootRef.value)
  // Re-measure once webfonts settle, otherwise the first paint may use fallback metrics.
  document.fonts?.ready.then(refresh)
})

onUnmounted(() => {
  observer?.disconnect()
  observer = null
})

watch(() => props.text, refresh)
watch(() => props.ellipsis, refresh)
</script>

<template>
  <span ref="rootRef" class="middle-ellipsis" :title="text">{{ display }}</span>
</template>

<style scoped>
.middle-ellipsis {
  @apply block overflow-hidden whitespace-nowrap;
}
</style>
