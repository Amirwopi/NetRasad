<template>
  <div class="stat-card card">
    <div class="stat-card__icon" :style="{ color: color }">
      <span v-html="icon" />
    </div>
    <div class="stat-card__body">
      <div class="stat-card__label muted">{{ label }}</div>
      <div class="stat-card__value mono">{{ value }}<span class="stat-card__unit dim">{{ unit }}</span></div>
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  label: string
  value: string | number
  unit?: string
  icon?: string
  color?: string
}>(), {
  unit: '',
  icon: '',
  color: 'var(--text)',
})
</script>

<style scoped>
.stat-card {
  display: flex; align-items: center; gap: 14px; padding: 16px;
}
.stat-card__icon { flex-shrink: 0; display: flex; align-items: center; }
.stat-card__label { font-size: 11px; text-transform: uppercase; letter-spacing: 0.05em; }
.stat-card__value { font-size: 22px; font-weight: 700; margin-top: 2px; }
.stat-card__unit { font-size: 13px; font-weight: 400; margin-left: 4px; }
</style>
