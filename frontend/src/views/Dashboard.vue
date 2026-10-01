<template>
  <div class="dashboard">
    <!-- Browser mode notice -->
    <div v-if="!traffic.isLive" class="notice">
      {{ $t('dashboard.browserMode') }}
    </div>

    <!-- Stat cards row -->
    <div class="grid grid-4">
      <StatCard
        :label="$t('dashboard.downloadRate')"
        :value="formatRate(traffic.downloadRate)"
        :icon="downloadIcon"
        color="var(--accent-down)"
      />
      <StatCard
        :label="$t('dashboard.uploadRate')"
        :value="formatRate(traffic.uploadRate)"
        :icon="uploadIcon"
        color="var(--accent-up)"
      />
      <StatCard
        :label="$t('dashboard.totalDownload')"
        :value="formatBytes(traffic.totalDownload)"
        :icon="totalDownIcon"
        color="var(--accent-down)"
      />
      <StatCard
        :label="$t('dashboard.totalUpload')"
        :value="formatBytes(traffic.totalUpload)"
        :icon="totalUpIcon"
        color="var(--accent-up)"
      />
    </div>

    <!-- Live traffic graph -->
    <TrafficGraph />

    <!-- Interfaces list -->
    <div class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('dashboard.interfaces') }}</span>
        <span class="muted" style="font-size: 12px">{{ traffic.interfaceList.length }} active</span>
      </div>
      <div class="card__body">
        <div v-if="traffic.interfaceList.length === 0" class="muted" style="text-align: center; padding: 20px">
          {{ $t('dashboard.noInterfaces') }}
        </div>
        <div v-else class="iface-list">
          <div v-for="iface in traffic.topInterfaces" :key="iface.interfaceId" class="iface-row">
            <div class="iface-row__name mono">
              {{ getInterfaceName(iface.interfaceId) }}
            </div>
            <div class="iface-row__stats">
              <span style="color: var(--accent-down)">↓ {{ formatRate(iface.downloadRate) }}</span>
              <span style="color: var(--accent-up)">↑ {{ formatRate(iface.uploadRate) }}</span>
              <span class="muted">{{ formatBytes(iface.downloadBytes + iface.uploadBytes) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Metrics (internal diagnostics) -->
    <div class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('dashboard.metrics') }}</span>
      </div>
      <div class="card__body">
        <div class="metrics-grid mono">
          <div><span class="muted">Samples Read:</span> {{ metrics?.samplesRead ?? '—' }}</div>
          <div><span class="muted">DB Writes:</span> {{ metrics?.dbWrites ?? '—' }}</div>
          <div><span class="muted">DB Errors:</span> {{ metrics?.dbErrors ?? '—' }}</div>
          <div><span class="muted">Dropped:</span> {{ metrics?.samplesDropped ?? '—' }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import StatCard from '@/components/StatCard.vue'
import TrafficGraph from '@/components/TrafficGraph.vue'
import { useTrafficStore } from '@/stores/traffic'
import { useFormat } from '@/composables/useFormat'
import { useWails } from '@/composables/useWails'
import type { Metrics } from '@/types'

const traffic = useTrafficStore()
const { formatRate, formatBytes } = useFormat()
const wails = useWails()

const metrics = ref<Metrics | null>(null)

const downloadIcon = '<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3v12m0 0l-4-4m4 4l4-4M5 21h14"/></svg>'
const uploadIcon = '<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 21V9m0 0l-4 4m4-4l4 4M5 3h14"/></svg>'
const totalDownIcon = '<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"/></svg>'
const totalUpIcon = '<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M17 8l-5-5-5 5M12 3v12"/></svg>'

function getInterfaceName(id: string): string {
  const iface = traffic.interfaces.find(i => i.id === id)
  if (!iface) return id
  const networkName = iface.ssid || iface.gatewayIP
  return networkName ? `${iface.name} (${networkName})` : iface.name
}

onMounted(async () => {
  try {
    metrics.value = await wails.getMetrics()
  } catch {  }
})
</script>

<style scoped>
.dashboard { display: flex; flex-direction: column; gap: 16px; }
.iface-list { display: flex; flex-direction: column; gap: 8px; }
.iface-row {
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 12px; border-radius: var(--radius-sm);
  background: var(--bg-elevated); transition: background var(--transition);
}
.iface-row:hover { background: var(--bg-card-hover); }
.iface-row__name { font-size: 13px; max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.iface-row__stats { display: flex; gap: 16px; font-size: 12px; }
.metrics-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; font-size: 13px; }
@media (max-width: 800px) { .metrics-grid { grid-template-columns: repeat(2, 1fr); } }
</style>
