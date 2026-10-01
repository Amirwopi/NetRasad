<template>
  <div class="quota-bar">
    <div class="quota-bar__header">
      <span class="muted">{{ label }}</span>
      <span class="mono">{{ formatBytes(used) }} / {{ formatBytes(limit) }}</span>
    </div>
    <div class="quota-bar__track">
      <div
        class="quota-bar__fill"
        :style="{ width: percent + '%', background: barColor }"
      />
    </div>
    <div class="quota-bar__percent mono" :style="{ color: barColor }">
      {{ percent.toFixed(1) }}%
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useFormat } from '@/composables/useFormat'

const props = defineProps<{
  label: string
  used: number
  limit: number
}>()

const { formatBytes } = useFormat()

const percent = computed(() => {
  if (props.limit <= 0) return 0
  return Math.min((props.used / props.limit) * 100, 100)
})

const barColor = computed(() => {
  if (percent.value >= 100) return 'var(--accent-error)'
  if (percent.value >= 90) return 'var(--accent-error)'
  if (percent.value >= 75) return 'var(--accent-warn)'
  return 'var(--accent-up)'
})
</script>

<style scoped>
.quota-bar { display: flex; flex-direction: column; gap: 6px; }
.quota-bar__header { display: flex; justify-content: space-between; font-size: 12px; }
.quota-bar__track {
  height: 8px; background: var(--bg-elevated); border-radius: 4px; overflow: hidden;
}
.quota-bar__fill { height: 100%; border-radius: 4px; transition: width 300ms ease; }
.quota-bar__percent { font-size: 13px; font-weight: 600; }
</style>
