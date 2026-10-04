<template>
  <div class="settings">
    <!-- General Settings -->
    <div class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('settings.general') }}</span>
      </div>
      <div class="card__body">
        <div class="settings-row">
          <label>{{ $t('settings.theme') }}</label>
          <select v-model="themeSelect" class="select" style="width: auto" @change="settings.setTheme(themeSelect)">
            <option value="dark">Dark</option>
            <option value="light">Light</option>
          </select>
        </div>
        <div class="settings-row">
          <label>{{ $t('settings.language') }}</label>
          <select v-model="localeSelect" class="select" style="width: auto" @change="settings.setLocale(localeSelect)">
            <option value="en">English</option>
            <option value="fa">فارسی</option>
          </select>
        </div>
      </div>
    </div>

    <!-- Graph Colours & Customization (Photo 4) -->
    <div class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('settings.graphColours') }}</span>
      </div>
      <div class="card__body grid-2-layout">
        <!-- Controls Column -->
        <div class="controls-col">
          <div class="settings-row">
            <label>{{ $t('settings.colorScheme') }}</label>
            <select v-model="schemeSelect" class="select" style="width: auto" @change="applyScheme">
              <option value="citrus">{{ $t('settings.citrus') }}</option>
              <option value="neon">{{ $t('settings.neon') }}</option>
              <option value="emerald">{{ $t('settings.emerald') }}</option>
              <option value="dark">{{ $t('settings.darkVelvet') }}</option>
              <option value="classic">{{ $t('settings.classic') }}</option>
              <option value="custom">{{ $t('settings.custom') }}</option>
            </select>
          </div>

          <div class="color-picker-row">
            <label>{{ $t('settings.bgColor') }}</label>
            <input type="color" v-model="s.graphBgColor" class="color-input" />
          </div>
          <div class="color-picker-row">
            <label>{{ $t('settings.dataIn') }}</label>
            <input type="color" v-model="s.graphDownColor" class="color-input" />
          </div>
          <div class="color-picker-row">
            <label>{{ $t('settings.dataOut') }}</label>
            <input type="color" v-model="s.graphUpColor" class="color-input" />
          </div>
          <div class="color-picker-row">
            <label>{{ $t('settings.textColor') }}</label>
            <input type="color" v-model="s.graphTextColor" class="color-input" />
          </div>

          <div class="settings-row">
            <label>{{ $t('settings.gradientBg') }}</label>
            <input type="checkbox" v-model="s.graphGradient" class="checkbox-input" />
          </div>

          <div class="slider-group">
            <label>{{ $t('settings.lineWidth') }}: <span class="mono">{{ s.graphLineWidth }}px</span></label>
            <input type="range" min="1" max="5" step="1" v-model.number="s.graphLineWidth" class="range-slider" />
          </div>
        </div>

        <!-- Live Sample Graph Preview Column (Photo 4) -->
        <div class="preview-col">
          <label class="muted preview-label">{{ $t('settings.sampleGraph') }}</label>
          <div class="sample-graph-box" :style="{ background: s.graphBgColor }">
            <canvas ref="sampleCanvasRef" width="300" height="150" />
            <div class="sample-legend mono" :style="{ color: s.graphTextColor }">
              <span :style="{ color: s.graphDownColor }">▲ 0.1 Mbit</span>
              <span :style="{ color: s.graphUpColor }">▼ 0.4 Mbit</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Taskbar / Speed Overlay Options -->
    <div class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('settings.extraStuff') }}</span>
      </div>
      <div class="card__body">
        <div class="settings-row">
          <label>{{ $t('taskbar.displayInTaskbar') }}</label>
          <input type="checkbox" v-model="s.displaySpeedInTaskbar" class="checkbox-input" @change="syncTaskbar" />
        </div>
        <div class="settings-row">
          <label>{{ $t('taskbar.transparent') }}</label>
          <input type="checkbox" v-model="s.taskbarTransparent" class="checkbox-input" @change="syncTaskbar" />
        </div>
        <div class="settings-row">
          <label>{{ $t('taskbar.position') }}</label>
          <select v-model="s.taskbarPosition" class="select" style="width: auto" @change="syncTaskbar">
            <option value="right">{{ $t('taskbar.positionRight') }}</option>
            <option value="left">{{ $t('taskbar.positionLeft') }}</option>
          </select>
        </div>
        <div class="settings-row">
          <label>{{ $t('taskbar.size') }}</label>
          <select v-model.number="s.taskbarWidth" class="select" style="width: auto" @change="syncTaskbar">
            <option :value="140">{{ $t('taskbar.sizeSmall') }}</option>
            <option :value="165">{{ $t('taskbar.sizeStandard') }}</option>
            <option :value="210">{{ $t('taskbar.sizeLarge') }}</option>
          </select>
        </div>
        <div class="settings-row">
          <label>Display Monitor</label>
          <select v-model.number="s.taskbarMonitor" class="select" style="width: auto" @change="syncTaskbar">
            <option :value="0">Primary Monitor</option>
            <option :value="1">Monitor 2</option>
            <option :value="2">Monitor 3</option>
            <option :value="3">Monitor 4</option>
          </select>
        </div>
        <div class="settings-row" v-if="s.taskbarPosition === 'left'">
          <label>Offset X (Pixels)</label>
          <input type="number" v-model.number="s.taskbarOffsetX" class="input" style="width: 100px" @change="syncTaskbar" />
        </div>
        <div class="settings-row">
          <label>{{ $t('settings.autoStart') }}</label>
          <input type="checkbox" v-model="s.autoStart" class="checkbox-input" @change="syncAutoStart" />
        </div>
      </div>
    </div>

    <!-- Ping Overlay Options -->
    <div class="card">
      <div class="card__header">
        <span class="card__title">Ping Overlay</span>
      </div>
      <div class="card__body">
        <div class="settings-row">
          <label>Enable Overlay (Always on top)</label>
          <input type="checkbox" v-model="s.pingOverlayEnabled" class="checkbox-input" />
        </div>
        <div class="settings-row">
          <label>Target IP/Domain</label>
          <input type="text" v-model="s.pingOverlayAddress" class="input" style="width: 150px" placeholder="8.8.8.8" />
        </div>
        <div class="settings-row">
          <label>Display Label</label>
          <input type="text" v-model="s.pingOverlayLabel" class="input" style="width: 150px" placeholder="Google" />
        </div>
        <div class="settings-row">
          <label>Widget Shape</label>
          <select v-model="s.pingOverlayShape" class="select" style="width: auto">
            <option value="rectangle">Rectangle (Rounded)</option>
            <option value="circle">Circle</option>
          </select>
        </div>
        <div class="color-picker-row">
          <label>Background Color</label>
          <input type="color" v-model="s.pingOverlayBgColorHex" class="color-input" />
        </div>
        <div class="color-picker-row">
          <label>Text Color</label>
          <input type="color" v-model="s.pingOverlayTextColorHex" class="color-input" />
        </div>
      </div>
    </div>

    <!-- Network Selection -->
    <div class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('settings.networkSelection') }}</span>
        <div v-if="activeInterfaces.length > 0" style="display: flex; gap: 8px;">
          <button class="btn btn-sm" @click="selectAll">{{ $t('settings.selectAll') }}</button>
          <button class="btn btn-sm" @click="deselectAll">{{ $t('settings.deselectAll') }}</button>
        </div>
      </div>
      <div class="card__body">
        <p class="muted" style="font-size: 12px; margin-bottom: 12px;">{{ $t('settings.networkSelectionDesc') }}</p>
        <div v-if="activeInterfaces.length === 0" class="muted" style="text-align: center; padding: 20px">
          {{ $t('settings.noActiveInterfaces') }}
        </div>
        <div v-else class="iface-select-list">
          <label v-for="iface in activeInterfaces" :key="iface.id" class="iface-select-row">
            <input type="checkbox" :value="iface.id" v-model="selectedIds" />
            <div class="iface-select-info">
              <span class="iface-select-name mono">{{ iface.name }}</span>
              <span class="iface-status-pill" :class="iface.isUp || iface.mediaConnected ? 'pill-online' : 'pill-offline'">
                {{ iface.isUp || iface.mediaConnected ? (iface.ssid ? `Connected: ${iface.ssid}` : 'Connected') : 'Disconnected' }}
              </span>
            </div>
            <div class="iface-select-details muted mono">
              <span v-if="iface.ipAddresses.length > 0">{{ iface.ipAddresses[0] }}</span>
              <span v-if="iface.gatewayIP">{{ $t('settings.gateway') }}: {{ iface.gatewayIP }}</span>
            </div>
          </label>
        </div>
        <div v-if="activeInterfaces.length > 0" class="settings-row" style="border-bottom: none; padding-top: 16px;">
          <button class="btn" @click="saveSelection" :disabled="saving">
            {{ saving ? 'Saving...' : 'Save' }}
          </button>
          <span v-if="saveResult" :style="{ color: saveResult === 'OK' ? 'var(--accent-up)' : 'var(--accent-error)' }" class="mono">
            {{ saveResult === 'OK' ? $t('settings.selectionSaved') : saveResult }}
          </span>
        </div>
      </div>
    </div>

    <!-- Database -->
    <div class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('settings.database') }}</span>
      </div>
      <div class="card__body">
        <div class="settings-row">
          <label class="muted">{{ $t('settings.dbPath') }}</label>
          <span class="mono">{{ dbPath }}</span>
        </div>
        <div class="settings-row">
          <button class="btn" @click="checkDb">{{ $t('settings.checkIntegrity') }}</button>
          <span v-if="dbCheckResult" :style="{ color: dbCheckResult === 'OK' ? 'var(--accent-up)' : 'var(--accent-error)' }" class="mono">
            {{ dbCheckResult }}
          </span>
        </div>
      </div>
    </div>

    <!-- About -->
    <div class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('settings.about') }}</span>
      </div>
      <div class="card__body">
        <div class="settings-row"><label class="muted">Version</label><span class="mono">0.1.0</span></div>
        <div class="settings-row"><label class="muted">Platform</label><span class="mono">{{ caps?.platform ?? '—' }}</span></div>
        <div class="settings-row"><label class="muted">Traffic Monitor</label><span :style="{color: caps?.trafficMonitor ? 'var(--accent-up)' : 'var(--accent-error)'}">{{ caps?.trafficMonitor ? '✓' : '✗' }}</span></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { useWails } from '@/composables/useWails'
import type { Capabilities, NetworkInterface, GraphColorScheme } from '@/types'

const settings = useSettingsStore()
const s = settings.settings
const wails = useWails()

const themeSelect = ref(s.theme)
const localeSelect = ref(s.locale)
const schemeSelect = ref<GraphColorScheme>(s.graphColorScheme)

const dbPath = ref('—')
const dbCheckResult = ref('')
const caps = ref<Capabilities | null>(null)

const activeInterfaces = ref<NetworkInterface[]>([])
const selectedIds = ref<string[]>([])
const saving = ref(false)
const saveResult = ref('')

const sampleCanvasRef = ref<HTMLCanvasElement | null>(null)

function applyScheme() {
  s.graphColorScheme = schemeSelect.value
  switch (schemeSelect.value) {
    case 'citrus':
      s.graphBgColor = '#3a3d42'
      s.graphDownColor = '#ffa500'
      s.graphUpColor = '#8bc34a'
      s.graphTextColor = '#ffffff'
      break
    case 'neon':
      s.graphBgColor = '#121824'
      s.graphDownColor = '#00d4ff'
      s.graphUpColor = '#00ff88'
      s.graphTextColor = '#e0e6ed'
      break
    case 'emerald':
      s.graphBgColor = '#18241e'
      s.graphDownColor = '#00e676'
      s.graphUpColor = '#b2ff59'
      s.graphTextColor = '#e8f5e9'
      break
    case 'dark':
      s.graphBgColor = '#181b20'
      s.graphDownColor = '#ff4081'
      s.graphUpColor = '#7c4dff'
      s.graphTextColor = '#e0e0e0'
      break
    case 'classic':
      s.graphBgColor = '#1e293b'
      s.graphDownColor = '#38bdf8'
      s.graphUpColor = '#4ade80'
      s.graphTextColor = '#f8fafc'
      break
  }
}

function drawSampleGraph() {
  const canvas = sampleCanvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  const w = canvas.width
  const h = canvas.height
  ctx.clearRect(0, 0, w, h)

  ctx.strokeStyle = 'rgba(255, 255, 255, 0.1)'
  ctx.lineWidth = 1
  for (let y = 30; y < h; y += 30) {
    ctx.beginPath()
    ctx.moveTo(0, y)
    ctx.lineTo(w, y)
    ctx.stroke()
  }

  const drawWave = (color: string, amplitude: number, offset: number) => {
    ctx.strokeStyle = color
    ctx.lineWidth = s.graphLineWidth
    ctx.beginPath()
    for (let x = 0; x <= w; x += 5) {
      const y = h / 2 + Math.sin((x + offset) * 0.04) * amplitude + Math.cos((x + offset) * 0.02) * 10
      if (x === 0) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
    }
    ctx.stroke()
    if (s.graphGradient) {
      ctx.lineTo(w, h)
      ctx.lineTo(0, h)
      ctx.closePath()
      const grad = ctx.createLinearGradient(0, 0, 0, h)
      grad.addColorStop(0, color + '44')
      grad.addColorStop(1, color + '00')
      ctx.fillStyle = grad
      ctx.fill()
    }
  }

  drawWave(s.graphDownColor, 25, 10)
  drawWave(s.graphUpColor, 15, 60)
}

watch([
  () => s.graphBgColor,
  () => s.graphDownColor,
  () => s.graphUpColor,
  () => s.graphGradient,
  () => s.graphLineWidth,
  () => s.taskbarPosition,
  () => s.taskbarWidth,
  () => s.taskbarTransparent,
  () => s.displaySpeedInTaskbar,
  () => s.pingOverlayEnabled,
  () => s.pingOverlayAddress,
  () => s.pingOverlayLabel,
  () => s.pingOverlayBgColorHex,
  () => s.pingOverlayTextColorHex,
  () => s.pingOverlayShape
], () => {
  drawSampleGraph()
  syncTaskbar()
  syncPingOverlay()
})

async function checkDb() {
  try {
    await wails.checkDatabase()
    dbCheckResult.value = 'OK'
  } catch (e: any) {
    dbCheckResult.value = e?.message ?? 'FAILED'
  }
}

async function loadInterfaces() {
  try {
    const list = await wails.getActiveInterfaces()
    activeInterfaces.value = list || []
    const stored = await wails.getSelectedInterfaces()
    if (!stored || stored.length === 0) {
      selectedIds.value = (list || []).filter(i => i.isUp || i.mediaConnected).map(i => i.id)
    } else {
      selectedIds.value = stored
    }
  } catch {  }
}

function selectAll() { selectedIds.value = activeInterfaces.value.map(i => i.id) }
function deselectAll() { selectedIds.value = [] }

async function saveSelection() {
  saving.value = true
  saveResult.value = ''
  try {
    await wails.setSelectedInterfaces(selectedIds.value)
    saveResult.value = 'OK'
    setTimeout(() => { saveResult.value = '' }, 3000)
  } catch (e: any) {
    saveResult.value = e?.message ?? 'FAILED'
  } finally {
    saving.value = false
  }
}

function syncTaskbar() {
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
}

function syncPingOverlay() {
  wails.setPingOverlayConfig({
    enabled: s.pingOverlayEnabled,
    address: s.pingOverlayAddress,
    label: s.pingOverlayLabel,
    bgColorHex: s.pingOverlayBgColorHex,
    textColorHex: s.pingOverlayTextColorHex,
    shape: s.pingOverlayShape,
    x: 0,
    y: 0
  })
}

async function syncAutoStart() {
  await wails.setAutoStart(s.autoStart)
}

onMounted(async () => {
  try {
    caps.value = await wails.getCapabilities()
    dbPath.value = caps.value?.dbPath ?? '—'
    s.autoStart = await wails.isAutoStartEnabled()
  } catch {  }
  await loadInterfaces()
  syncTaskbar()
  setTimeout(drawSampleGraph, 100)
})
</script>

<style scoped>
.settings { display: flex; flex-direction: column; gap: 16px; max-width: 760px; }
.settings-row { display: flex; align-items: center; justify-content: space-between; padding: 10px 0; border-bottom: 1px solid var(--border); }
.settings-row:last-child { border-bottom: none; }
.color-picker-row { display: flex; align-items: center; justify-content: space-between; padding: 6px 0; font-size: 13px; }
.color-input { width: 36px; height: 26px; border: none; border-radius: 4px; cursor: pointer; background: transparent; }
.checkbox-input { width: 18px; height: 18px; cursor: pointer; }
.slider-group { display: flex; flex-direction: column; gap: 6px; padding: 8px 0; font-size: 13px; }
.range-slider { width: 100%; cursor: pointer; }
.grid-2-layout { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; }
.sample-graph-box { position: relative; border-radius: var(--radius-sm); border: 1px solid var(--border); padding: 10px; display: flex; flex-direction: column; align-items: center; justify-content: center; }
.preview-label { font-size: 12px; margin-bottom: 8px; display: block; }
.sample-legend { display: flex; gap: 16px; font-size: 12px; font-weight: 600; margin-top: 8px; }
.iface-select-list { display: flex; flex-direction: column; gap: 4px; }
.iface-select-row { display: flex; align-items: center; gap: 12px; padding: 10px 12px; border-radius: var(--radius-sm); background: var(--bg-elevated); cursor: pointer; }
.iface-select-info { display: flex; align-items: baseline; gap: 6px; flex: 1; }
.iface-select-name { font-size: 13px; }
.iface-select-network { font-size: 12px; }
.iface-select-details { display: flex; gap: 16px; font-size: 11px; }
.iface-status-pill { font-size: 11px; padding: 2px 8px; border-radius: 10px; font-family: monospace; font-weight: 600; }
.pill-online { background: rgba(0, 255, 136, 0.12); color: var(--accent-up, #00ff88); border: 1px solid rgba(0, 255, 136, 0.3); }
.pill-offline { background: rgba(255, 255, 255, 0.05); color: var(--text-muted, #8b95a5); border: 1px solid rgba(255, 255, 255, 0.1); }
.btn-sm { font-size: 11px; padding: 4px 10px; }
</style>
