<template>
  <div class="traffic-graph card">
    <div class="card__header">
      <span class="card__title">{{ $t('dashboard.liveGraph') }}</span>
      <div class="traffic-graph__legend">
        <span class="traffic-graph__legend-item">
          <span class="traffic-graph__dot" style="background: var(--accent-down)" />
          {{ $t('dashboard.download') }} <span class="mono">{{ formatRate(currentDown) }}</span>
        </span>
        <span class="traffic-graph__legend-item">
          <span class="traffic-graph__dot" style="background: var(--accent-up)" />
          {{ $t('dashboard.upload') }} <span class="mono">{{ formatRate(currentUp) }}</span>
        </span>
      </div>
    </div>
    <div class="traffic-graph__canvas-wrap">
      <canvas ref="canvasRef" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useTrafficStore } from '@/stores/traffic'
import { useSettingsStore } from '@/stores/settings'
import { useFormat } from '@/composables/useFormat'

const traffic = useTrafficStore()
const settingsStore = useSettingsStore()
const s = settingsStore.settings
const { formatRate } = useFormat()

const canvasRef = ref<HTMLCanvasElement | null>(null)
let rafId = 0
let resizeObserver: ResizeObserver | null = null

const currentDown = ref(0)
const currentUp = ref(0)

function draw() {
  const canvas = canvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  const dpr = window.devicePixelRatio || 1
  const rect = canvas.getBoundingClientRect()
  const w = rect.width
  const h = rect.height

  if (canvas.width !== w * dpr || canvas.height !== h * dpr) {
    canvas.width = w * dpr
    canvas.height = h * dpr
    ctx.scale(dpr, dpr)
  }

  const c = ctx
  c.clearRect(0, 0, w, h)

  c.fillStyle = 'transparent'
  c.fillRect(0, 0, w, h)

  const samples = traffic.rateHistory
  if (samples.length < 2) {
    c.fillStyle = 'var(--text-dim)'
    c.font = '13px var(--font-sans)'
    c.textAlign = 'center'
    c.fillText('Waiting for data...', w / 2, h / 2)
    rafId = requestAnimationFrame(draw)
    return
  }

  let maxRate = 1
  for (const s of samples) {
    if (s.downloadBps > maxRate) maxRate = s.downloadBps
    if (s.uploadBps > maxRate) maxRate = s.uploadBps
  }
  maxRate *= 1.15

  const padLeft = 50
  const padRight = 12
  const padTop = 12
  const padBottom = 24
  const graphW = w - padLeft - padRight
  const graphH = h - padTop - padBottom

  c.strokeStyle = getCssVar('--border') || '#2a3142'
  c.lineWidth = 1
  c.font = '10px monospace'
  c.fillStyle = getCssVar('--text-dim') || '#5c6675'
  c.textAlign = 'right'
  c.textBaseline = 'middle'

  const gridLines = 5
  for (let i = 0; i <= gridLines; i++) {
    const y = padTop + (graphH / gridLines) * i
    c.beginPath()
    c.moveTo(padLeft, y)
    c.lineTo(w - padRight, y)
    c.stroke()
    const val = maxRate * (1 - i / gridLines)
    c.fillText(formatRate(val), padLeft - 6, y)
  }

  c.textAlign = 'center'
  c.textBaseline = 'top'
  const timeSteps = 5
  for (let i = 0; i <= timeSteps; i++) {
    const x = padLeft + (graphW / timeSteps) * i
    const idx = Math.floor((samples.length - 1) * (i / timeSteps))
    if (idx >= 0 && idx < samples.length) {
      const ts = new Date(samples[idx].timestamp)
      const label = `${ts.getMinutes().toString().padStart(2, '0')}:${ts.getSeconds().toString().padStart(2, '0')}`
      c.fillText(label, x, h - padBottom + 6)
    }
  }

  const n = samples.length
  const xStep = graphW / Math.max(n - 1, 1)

  const downColor = s.graphDownColor || getCssVar('--accent-down') || '#00d4ff'
  const upColor = s.graphUpColor || getCssVar('--accent-up') || '#00ff88'
  const lineWidth = s.graphLineWidth || 2
  const fillGradient = s.graphGradient ?? true

  function drawLine(getY: (s: typeof samples[0]) => number, color: string, fill: boolean) {
    c.strokeStyle = color
    c.lineWidth = lineWidth
    c.lineJoin = 'round'
    c.beginPath()
    for (let i = 0; i < n; i++) {
      const x = padLeft + xStep * i
      const y = padTop + graphH * (1 - getY(samples[i]) / maxRate)
      if (i === 0) c.moveTo(x, y)
      else c.lineTo(x, y)
    }
    c.stroke()

    if (fill && fillGradient) {
      c.lineTo(padLeft + xStep * (n - 1), padTop + graphH)
      c.lineTo(padLeft, padTop + graphH)
      c.closePath()
      const grad = c.createLinearGradient(0, padTop, 0, padTop + graphH)
      grad.addColorStop(0, color + '33')
      grad.addColorStop(1, color + '00')
      c.fillStyle = grad
      c.fill()
    }
  }

  drawLine(s => s.downloadBps, downColor, true)
  drawLine(s => s.uploadBps, upColor, true)

  const last = samples[samples.length - 1]
  currentDown.value = last.downloadBps
  currentUp.value = last.uploadBps

  rafId = requestAnimationFrame(draw)
}

function getCssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || ''
}

onMounted(() => {
  resizeObserver = new ResizeObserver(() => {
  })
  if (canvasRef.value) resizeObserver.observe(canvasRef.value)
  rafId = requestAnimationFrame(draw)
})

onUnmounted(() => {
  cancelAnimationFrame(rafId)
  resizeObserver?.disconnect()
})
</script>

<style scoped>
.traffic-graph__canvas-wrap {
  padding: 0 12px 12px 12px;
}
.traffic-graph__canvas-wrap canvas {
  width: 100%;
  height: 240px;
  display: block;
}
.traffic-graph__legend {
  display: flex; gap: 16px; font-size: 12px;
}
.traffic-graph__legend-item {
  display: flex; align-items: center; gap: 6px; color: var(--text-muted);
}
.traffic-graph__dot {
  width: 8px; height: 8px; border-radius: 50%;
}
</style>
