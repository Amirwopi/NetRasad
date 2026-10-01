<template>
  <div class="app-shell">
    <Sidebar />
    <div class="main-area">
      <header class="main-header">
        <span class="main-header__title">{{ $t('app.name') }}</span>
        <span class="main-header__meta mono">
          <span v-if="traffic.isLive" style="color: var(--accent-up)">● </span>
          <span v-else style="color: var(--text-dim)">○ </span>
          {{ formatRate(traffic.downloadRate) }} ↓ · {{ formatRate(traffic.uploadRate) }} ↑
        </span>
      </header>
      <main class="main-content">
        <router-view />
      </main>
    </div>
    <SpeedWidget />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import Sidebar from './components/Sidebar.vue'
import SpeedWidget from './components/SpeedWidget.vue'
import { useTrafficStore } from './stores/traffic'
import { useSettingsStore } from './stores/settings'
import { useFormat } from './composables/useFormat'
import { useWails } from './composables/useWails'

const traffic = useTrafficStore()
const settings = useSettingsStore()
const { formatRate } = useFormat()
const wails = useWails()

let unsub: (() => void) | null = null

onMounted(async () => {
  settings.load()
  try {
    const [snap, rates, ifaces] = await Promise.all([
      wails.getDashboardState(),
      wails.getRateHistory(),
      wails.getInterfaces(),
    ])
    if (snap) traffic.setSnapshot(snap)
    if (rates) traffic.setRateHistory(rates)
    if (ifaces) traffic.setInterfaces(ifaces)
  } catch {
    traffic.setLive(false)
  }
  unsub = wails.onTrafficUpdated((data) => {
    traffic.setSnapshot(data)
  })
})

onUnmounted(() => {
  unsub?.()
})
</script>
