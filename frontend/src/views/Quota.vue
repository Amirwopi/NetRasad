<template>
  <div class="quota-page">
    <div class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('quota.title') }}</span>
        <span class="badge" :class="statusBadgeClass">{{ statusText }}</span>
      </div>
      <div class="card__body">
        <!-- Consumption Progress -->
        <div class="quota-progress-section">
          <div class="usage-summary">
            <div class="stat-box">
              <span class="label">{{ $t('quota.usage') }}</span>
              <span class="value mono" :style="{ color: statusColor }">{{ formatBytes(currentUsage) }}</span>
            </div>
            <div class="stat-box">
              <span class="label">{{ $t('quota.limit') }}</span>
              <span class="value mono">{{ formatBytes(quotaConfig.limitBytes) }}</span>
            </div>
            <div class="stat-box">
              <span class="label">Remaining</span>
              <span class="value mono">{{ formatBytes(Math.max(0, quotaConfig.limitBytes - currentUsage)) }}</span>
            </div>
          </div>

          <div class="progress-bar-wrap">
            <div class="progress-bar-bg">
              <div
                class="progress-bar-fill"
                :style="{ width: `${usagePercent}%`, background: statusColor }"
              />
            </div>
            <div class="progress-labels mono">
              <span>{{ usagePercent.toFixed(1) }}% used</span>
              <span>Warning at {{ quotaConfig.warningPercent }}%</span>
            </div>
          </div>
        </div>

        <!-- Quota Settings Configuration Form -->
        <div class="quota-config-card">
          <h4 class="section-title">{{ $t('quota.setQuota') }}</h4>
          <div class="form-grid">
            <div class="form-group">
              <label>{{ $t('quota.limit') }} (GB)</label>
              <input
                type="number"
                v-model.number="limitGB"
                class="input"
                min="1"
                max="10000"
              />
            </div>
            <div class="form-group">
              <label>{{ $t('quota.period') }}</label>
              <select v-model="quotaConfig.period" class="select">
                <option value="daily">{{ $t('quota.daily') }}</option>
                <option value="weekly">{{ $t('quota.weekly') }}</option>
                <option value="monthly">{{ $t('quota.monthly') }}</option>
              </select>
            </div>
            <div class="form-group">
              <label>{{ $t('quota.warning') }} (%)</label>
              <input
                type="number"
                v-model.number="quotaConfig.warningPercent"
                class="input"
                min="50"
                max="99"
              />
            </div>
          </div>
          <div class="action-row">
            <button class="btn btn--primary" @click="saveQuota">{{ $t('quota.save') }}</button>
            <span v-if="savedMsg" class="mono saved-text">Saved ✓</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { useTrafficStore } from '@/stores/traffic'
import { useFormat } from '@/composables/useFormat'

const settings = useSettingsStore()
const traffic = useTrafficStore()
const { formatBytes } = useFormat()

const quotaConfig = ref({ ...settings.settings.quota })
const limitGB = ref(Math.round(quotaConfig.value.limitBytes / (1024 * 1024 * 1024)))
const savedMsg = ref(false)

const currentUsage = computed(() => traffic.totalDownload + traffic.totalUpload)

const usagePercent = computed(() => {
  if (quotaConfig.value.limitBytes <= 0) return 0
  return Math.min(100, (currentUsage.value / quotaConfig.value.limitBytes) * 100)
})

const statusColor = computed(() => {
  if (usagePercent.value >= 100) return 'var(--accent-error, #ff4444)'
  if (usagePercent.value >= quotaConfig.value.warningPercent) return 'var(--accent-warn, #ffa500)'
  return 'var(--accent-up, #00ff88)'
})

const statusBadgeClass = computed(() => {
  if (usagePercent.value >= 100) return 'badge--danger'
  if (usagePercent.value >= quotaConfig.value.warningPercent) return 'badge--warn'
  return 'badge--success'
})

const statusText = computed(() => {
  if (usagePercent.value >= 100) return 'Exceeded'
  if (usagePercent.value >= quotaConfig.value.warningPercent) return 'Warning'
  return 'Normal'
})

function saveQuota() {
  quotaConfig.value.limitBytes = limitGB.value * 1024 * 1024 * 1024
  settings.settings.quota = { ...quotaConfig.value }
  savedMsg.value = true
  setTimeout(() => { savedMsg.value = false }, 3000)
}
</script>

<style scoped>
.quota-page { display: flex; flex-direction: column; gap: 16px; max-width: 700px; }
.badge { font-size: 11px; font-weight: 700; padding: 3px 8px; border-radius: 4px; text-transform: uppercase; }
.badge--success { background: rgba(0, 255, 136, 0.15); color: var(--accent-up); }
.badge--warn { background: rgba(255, 165, 0, 0.15); color: var(--accent-warn); }
.badge--danger { background: rgba(255, 68, 68, 0.15); color: var(--accent-error); }
.usage-summary { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin-bottom: 20px; }
.stat-box { display: flex; flex-direction: column; gap: 4px; padding: 12px; border-radius: var(--radius-sm); background: var(--bg-elevated); }
.stat-box .label { font-size: 11px; color: var(--text-muted); text-transform: uppercase; }
.stat-box .value { font-size: 20px; font-weight: 700; }
.progress-bar-wrap { display: flex; flex-direction: column; gap: 6px; margin-bottom: 24px; }
.progress-bar-bg { height: 14px; border-radius: 7px; background: var(--bg-input); border: 1px solid var(--border); overflow: hidden; }
.progress-bar-fill { height: 100%; transition: width 300ms ease; }
.progress-labels { display: flex; justify-content: space-between; font-size: 12px; color: var(--text-muted); }
.quota-config-card { border-top: 1px solid var(--border); padding-top: 16px; }
.section-title { font-size: 14px; margin-bottom: 12px; }
.form-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin-bottom: 16px; }
.form-group { display: flex; flex-direction: column; gap: 6px; }
.form-group label { font-size: 12px; color: var(--text-muted); }
.action-row { display: flex; align-items: center; gap: 12px; }
.saved-text { color: var(--accent-up); font-size: 13px; }
</style>
