<template>
  <div class="applications">
    <div class="card">
      <div class="card__header">
        <div class="title-area">
          <span class="card__title">{{ $t('applications.title') }}</span>
          <span class="muted" style="font-size: 12px">{{ filteredProcesses.length }} {{ $t('applications.process') }}</span>
        </div>
        <div class="toolbar">
          <input
            v-model="search"
            type="text"
            class="input search-input"
            :placeholder="$t('applications.searchPlaceholder')"
          />
          <div class="view-switch">
            <button
              class="btn btn-sm"
              :class="{ 'btn--primary': viewMode === 'graph' }"
              @click="viewMode = 'graph'"
              title="Graph View"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M3 3v18h18"/><path d="M18 17V9"/><path d="M13 17V5"/><path d="M8 17v-3"/>
              </svg>
              <span>{{ $t('applications.graphView') }}</span>
            </button>
            <button
              class="btn btn-sm"
              :class="{ 'btn--primary': viewMode === 'table' }"
              @click="viewMode = 'table'"
              title="Table View"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M3 15h18"/><path d="M9 3v18"/>
              </svg>
              <span>{{ $t('applications.tableView') }}</span>
            </button>
          </div>
        </div>
      </div>
      <div class="card__body">
        <!-- Graph View -->
        <div v-show="viewMode === 'graph'" class="graph-container">
          <div class="graph-canvas-wrap">
            <canvas ref="canvasRef" />
          </div>
        </div>

        <!-- Table View -->
        <div v-show="viewMode === 'table'" class="table-container">
          <div v-if="filteredProcesses.length === 0" class="muted empty-box">
            {{ $t('applications.noProcesses') }}
          </div>
          <table v-else class="app-table">
            <thead>
              <tr>
                <th @click="sortBy('processName')" class="sortable">
                  {{ $t('applications.process') }}
                  <span v-if="sortKey === 'processName'">{{ sortOrder > 0 ? '▲' : '▼' }}</span>
                </th>
                <th @click="sortBy('pid')" class="sortable">
                  {{ $t('applications.pid') }}
                  <span v-if="sortKey === 'pid'">{{ sortOrder > 0 ? '▲' : '▼' }}</span>
                </th>
                <th @click="sortBy('downloadBps')" class="sortable">
                  {{ $t('applications.downloadSpeed') }}
                  <span v-if="sortKey === 'downloadBps'">{{ sortOrder > 0 ? '▲' : '▼' }}</span>
                </th>
                <th @click="sortBy('uploadBps')" class="sortable">
                  {{ $t('applications.uploadSpeed') }}
                  <span v-if="sortKey === 'uploadBps'">{{ sortOrder > 0 ? '▲' : '▼' }}</span>
                </th>
                <th @click="sortBy('totalDownload')" class="sortable">
                  {{ $t('applications.totalData') }}
                  <span v-if="sortKey === 'totalDownload'">{{ sortOrder > 0 ? '▲' : '▼' }}</span>
                </th>
                <th @click="sortBy('connCount')" class="sortable">
                  {{ $t('applications.connections') }}
                  <span v-if="sortKey === 'connCount'">{{ sortOrder > 0 ? '▲' : '▼' }}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in filteredProcesses" :key="p.pid">
                <td class="app-name-cell">
                  <div class="app-icon">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect x="4" y="4" width="16" height="16" rx="2"/>
                      <circle cx="9" cy="9" r="2"/>
                      <path d="M15 15l-3-3"/>
                    </svg>
                  </div>
                  <span class="mono bold">{{ p.processName }}</span>
                </td>
                <td class="mono dim">{{ p.pid }}</td>
                <td class="mono down-text">↓ {{ formatRate(p.downloadBps) }}</td>
                <td class="mono up-text">↑ {{ formatRate(p.uploadBps) }}</td>
                <td class="mono">{{ formatBytes(p.totalDownload + p.totalUpload) }}</td>
                <td class="mono text-center">{{ p.connCount }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useWails } from '@/composables/useWails'
import { useFormat } from '@/composables/useFormat'
import type { ProcessTraffic } from '@/types'

const wails = useWails()
const { formatRate, formatBytes } = useFormat()

const processes = ref<ProcessTraffic[]>([])
const search = ref('')
const viewMode = ref<'graph' | 'table'>('table')
const sortKey = ref<keyof ProcessTraffic>('downloadBps')
const sortOrder = ref(-1)

let unsub: (() => void) | null = null
const canvasRef = ref<HTMLCanvasElement | null>(null)
let animationRunning = false
let rafId = 0

function loadMock() {
  processes.value = [
    { pid: 4812, processName: 'chrome.exe', downloadBps: 3420000, uploadBps: 120000, totalDownload: 480000000, totalUpload: 34000000, connCount: 14 },
    { pid: 8192, processName: 'firefox.exe', downloadBps: 1540000, uploadBps: 45000, totalDownload: 210000000, totalUpload: 18000000, connCount: 8 },
    { pid: 1042, processName: 'svchost.exe', downloadBps: 52000, uploadBps: 1200, totalDownload: 12000000, totalUpload: 450000, connCount: 5 },
    { pid: 6320, processName: 'telegram.exe', downloadBps: 18000, uploadBps: 3400, totalDownload: 85000000, totalUpload: 9200000, connCount: 4 },
  ]
}

const filteredProcesses = computed(() => {
  let list = processes.value
  if (search.value.trim()) {
    const q = search.value.toLowerCase().trim()
    list = list.filter(p => p.processName.toLowerCase().includes(q) || String(p.pid).includes(q))
  }
  const key = sortKey.value
  return [...list].sort((a, b) => {
    const av = a[key] ?? 0
    const bv = b[key] ?? 0
    if (av < bv) return -1 * sortOrder.value
    if (av > bv) return 1 * sortOrder.value
    return 0
  })
})

function sortBy(key: keyof ProcessTraffic) {
  if (sortKey.value === key) {
    sortOrder.value *= -1
  } else {
    sortKey.value = key
    sortOrder.value = -1
  }
}

async function fetchProcesses() {
  try {
    const list = await wails.getProcessTraffic()
    if (list && list.length > 0) processes.value = list
    else if (processes.value.length === 0) loadMock()
  } catch {
    if (processes.value.length === 0) loadMock()
  }
}

function drawGraph() {
  if (!animationRunning) return
  const canvas = canvasRef.value
  if (!canvas) {
    rafId = requestAnimationFrame(drawGraph)
    return
  }
  const ctx = canvas.getContext('2d')
  if (!ctx) {
    rafId = requestAnimationFrame(drawGraph)
    return
  }

  const dpr = window.devicePixelRatio || 1
  const rect = canvas.getBoundingClientRect()
  if (rect.width === 0 || rect.height === 0) {
    rafId = requestAnimationFrame(drawGraph)
    return
  }

  canvas.width = rect.width * dpr
  canvas.height = rect.height * dpr
  ctx.scale(dpr, dpr)

  const w = rect.width
  const h = rect.height
  ctx.clearRect(0, 0, w, h)

  const top = filteredProcesses.value.slice(0, 7)
  if (top.length === 0) {
    ctx.fillStyle = '#64748b'
    ctx.font = '13px sans-serif'
    ctx.textAlign = 'center'
    ctx.fillText('No process network activity', w / 2, h / 2)
    rafId = requestAnimationFrame(drawGraph)
    return
  }

  const maxRate = Math.max(...top.map(p => p.downloadBps + p.uploadBps), 1024)
  const barH = 26
  const gap = 16
  const startY = 16
  const labelW = 150

  top.forEach((p, i) => {
    const y = startY + i * (barH + gap)
    if (y + barH > h) return

    ctx.fillStyle = '#cbd5e1'
    ctx.font = '12px monospace'
    ctx.textAlign = 'left'
    const dispName = p.processName.length > 18 ? p.processName.slice(0, 16) + '…' : p.processName
    ctx.fillText(`${dispName} [${p.pid}]`, 10, y + 17)

    const barW = Math.max(w - labelW - 140, 50)
    ctx.fillStyle = 'rgba(255, 255, 255, 0.04)'
    ctx.beginPath()
    ctx.roundRect ? ctx.roundRect(labelW, y, barW, barH, 4) : ctx.fillRect(labelW, y, barW, barH)
    ctx.fill()

    const downW = (p.downloadBps / maxRate) * barW
    if (downW > 0) {
      const gradDown = ctx.createLinearGradient(labelW, 0, labelW + downW, 0)
      gradDown.addColorStop(0, '#00b4d8')
      gradDown.addColorStop(1, '#00d4ff')
      ctx.fillStyle = gradDown
      ctx.beginPath()
      ctx.roundRect ? ctx.roundRect(labelW, y, downW, barH, 4) : ctx.fillRect(labelW, y, downW, barH)
      ctx.fill()
    }

    ctx.fillStyle = '#38bdf8'
    ctx.font = '11px monospace'
    ctx.textAlign = 'left'
    ctx.fillText(`↓ ${formatRate(p.downloadBps)}`, labelW + downW + 8, y + 17)
  })

  rafId = requestAnimationFrame(drawGraph)
}

function startGraph() {
  if (!animationRunning) {
    animationRunning = true
    nextTick(() => {
      drawGraph()
    })
  }
}

function stopGraph() {
  animationRunning = false
  cancelAnimationFrame(rafId)
}

watch(viewMode, (newVal) => {
  if (newVal === 'graph') {
    startGraph()
  } else {
    stopGraph()
  }
})

onMounted(async () => {
  await fetchProcesses()
  unsub = wails.onProcessTrafficUpdated((data) => {
    if (data && data.length > 0) processes.value = data
  })
  if (viewMode.value === 'graph') {
    startGraph()
  }
})

onUnmounted(() => {
  unsub?.()
  stopGraph()
})
</script>

<style scoped>
.applications { display: flex; flex-direction: column; gap: 16px; }
.title-area { display: flex; align-items: center; gap: 10px; }
.toolbar { display: flex; align-items: center; gap: 12px; }
.search-input { width: 220px; }
.view-switch { display: flex; gap: 4px; background: var(--bg-input); padding: 3px; border-radius: var(--radius-sm); border: 1px solid var(--border); }
.btn-sm { padding: 4px 10px; font-size: 12px; border: none; background: transparent; }
.btn--primary { background: var(--accent-down); color: #001018; }
.table-container { overflow-x: auto; }
.app-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.app-table th { text-align: left; padding: 10px 12px; border-bottom: 1px solid var(--border); color: var(--text-muted); font-weight: 600; font-size: 11px; text-transform: uppercase; }
.app-table td { padding: 10px 12px; border-bottom: 1px solid var(--border); }
.app-table tr:hover td { background: var(--bg-card-hover); }
.sortable { cursor: pointer; user-select: none; }
.sortable:hover { color: var(--accent-down); }
.app-name-cell { display: flex; align-items: center; gap: 10px; }
.app-icon { width: 28px; height: 28px; border-radius: 6px; background: var(--bg-elevated); display: flex; align-items: center; justify-content: center; color: var(--accent-down); }
.bold { font-weight: 600; color: var(--text); }
.down-text { color: var(--accent-down); }
.up-text { color: var(--accent-up); }
.empty-box { text-align: center; padding: 40px; }
.graph-container { padding: 12px 0; min-height: 320px; }
.graph-canvas-wrap canvas { width: 100%; height: 320px; display: block; }
</style>
