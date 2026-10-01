<template>
  <div
    v-if="s.displaySpeedInTaskbar"
    class="speed-widget-taskbar"
    :class="{ 'speed-widget-taskbar--transparent': s.taskbarTransparent }"
  >
    <div class="taskbar-band mono">
      <span class="rate-item down-text" title="Download Speed">
        <svg class="speed-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <path d="M12 3v12m0 0l-4-4m4 4l4-4M5 21h14"/>
        </svg>
        <span class="speed-val">{{ formatRate(traffic.downloadRate) }}</span>
      </span>
      <span class="divider">|</span>
      <span class="rate-item up-text" title="Upload Speed">
        <svg class="speed-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <path d="M12 21V9m0 0l-4 4m4-4l4 4M5 3h14"/>
        </svg>
        <span class="speed-val">{{ formatRate(traffic.uploadRate) }}</span>
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useTrafficStore } from '@/stores/traffic'
import { useSettingsStore } from '@/stores/settings'
import { useFormat } from '@/composables/useFormat'

const traffic = useTrafficStore()
const settingsStore = useSettingsStore()
const s = settingsStore.settings
const { formatRate } = useFormat()
</script>

<style scoped>
.speed-widget-taskbar {
  position: fixed;
  bottom: 10px;
  right: 20px;
  z-index: 999999;
  background: rgba(18, 22, 28, 0.88);
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 20px;
  padding: 4px 14px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5), inset 0 1px 1px rgba(255, 255, 255, 0.1);
  display: flex;
  align-items: center;
  height: 30px;
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  user-select: none;
  transition: all 0.2s ease;
}

html[dir='rtl'] .speed-widget-taskbar {
  right: auto;
  left: 20px;
}

.speed-widget-taskbar:hover {
  background: rgba(24, 30, 40, 0.95);
  border-color: rgba(0, 212, 255, 0.4);
  box-shadow: 0 8px 28px rgba(0, 212, 255, 0.15);
}

.speed-widget-taskbar--transparent {
  background: rgba(12, 16, 22, 0.6);
  border-color: rgba(255, 255, 255, 0.08);
}

.taskbar-band {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.3px;
}

.rate-item {
  display: flex;
  align-items: center;
  gap: 5px;
}

.speed-icon {
  flex-shrink: 0;
}

.speed-val {
  white-space: nowrap;
}

.divider {
  color: rgba(255, 255, 255, 0.2);
  font-size: 10px;
}

.down-text {
  color: var(--accent-down, #00d4ff);
}

.up-text {
  color: var(--accent-up, #00ff88);
}
</style>
