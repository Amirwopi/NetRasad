<template>
  <div class="app-shell">
    <Sidebar />
    <div class="main-area">
      <header class="main-header">
        <span class="main-header__title">{{ $t('app.name') }}</span>
        <div class="header-speed-widget mono">
          <span v-if="traffic.isLive" style="color: var(--accent-up); margin-right: 8px;">●</span>
          <span v-else style="color: var(--text-dim); margin-right: 8px;">○</span>
          <span class="rate-item down-text" title="Download Speed">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M12 3v12m0 0l-4-4m4 4l4-4M5 21h14"/>
            </svg>
            <span class="speed-val">{{ formatRate(traffic.downloadRate) }}</span>
          </span>
          <span class="divider">|</span>
          <span class="rate-item up-text" title="Upload Speed">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M12 21V9m0 0l-4 4m4-4l4 4M5 3h14"/>
            </svg>
            <span class="speed-val">{{ formatRate(traffic.uploadRate) }}</span>
          </span>
        </div>
      </header>
      <main class="main-content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import Sidebar from './components/Sidebar.vue'
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
  const s = settings.settings
  
  // Apply Taskbar Settings on Boot
  try {
    wails.setTaskbarWidgetEnabled(s.displaySpeedInTaskbar)
    wails.setTaskbarConfig({
      downColorHex: s.graphDownColor,
      upColorHex: s.graphUpColor,
      bgColorHex: s.taskbarTransparent ? 'transparent' : s.graphBgColor,
      transparentBg: s.taskbarTransparent,
      position: s.taskbarPosition || 'right',
      width: s.taskbarWidth || 165,
      offsetX: s.taskbarOffsetX || 0,
      monitor: s.taskbarMonitor || 0,
    })
  } catch(e) {}
  
  // Apply Ping Overlay Settings on Boot
  try {
    const currentCfg = await wails.getPingOverlayConfig()
    if (currentCfg && currentCfg.x !== 0 && currentCfg.y !== 0) {
      s.pingOverlayX = currentCfg.x
      s.pingOverlayY = currentCfg.y
    }
    wails.setPingOverlayConfig({
      enabled: s.pingOverlayEnabled,
      address: s.pingOverlayAddress,
      label: s.pingOverlayLabel,
      bgColorHex: s.pingOverlayBgColorHex,
      textColorHex: s.pingOverlayTextColorHex,
      shape: s.pingOverlayShape,
      x: s.pingOverlayX,
      y: s.pingOverlayY
    })
    setInterval(async () => {
      if (s.pingOverlayEnabled) {
        try {
          const cfg = await wails.getPingOverlayConfig()
          if (cfg && (cfg.x !== s.pingOverlayX || cfg.y !== s.pingOverlayY)) {
            if (cfg.x !== 0 || cfg.y !== 0) {
              s.pingOverlayX = cfg.x
              s.pingOverlayY = cfg.y
            }
          }
        } catch(e) {}
      }
    }, 3000)
  } catch(e) {}

  try {
    const { GetHistoricalUsage } = await import('../wailsjs/go/app/App')
    // Get past 30 days default
    const startDate = new Date()
    startDate.setDate(startDate.getDate() - 30)
    const historical = await GetHistoricalUsage(startDate.toISOString())
    traffic.historicalDownload = historical.download || 0
    traffic.historicalUpload = historical.upload || 0
  } catch(e) {}

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
