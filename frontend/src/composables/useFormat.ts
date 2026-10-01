const BINARY_UNITS = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB']
const DECIMAL_UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']

function pickBase(units: 'binary' | 'decimal'): number {
  return units === 'binary' ? 1024 : 1000
}

function pickLabels(units: 'binary' | 'decimal'): string[] {
  return units === 'binary' ? BINARY_UNITS : DECIMAL_UNITS
}

function safeNumber(n: number): number {
  if (!Number.isFinite(n) || n < 0) return 0
  return n
}

export function formatBytes(
  bytes: number,
  units: 'binary' | 'decimal' = 'binary',
): string {
  const b = safeNumber(bytes)
  const base = pickBase(units)
  const labels = pickLabels(units)
  if (b < base) return `${b.toFixed(0)} ${labels[0]}`
  const i = Math.min(
    Math.floor(Math.log(b) / Math.log(base)),
    labels.length - 1,
  )
  const val = b / Math.pow(base, i)
  return `${val.toFixed(2)} ${labels[i]}`
}

export function formatRate(
  bytesPerSec: number,
  units: 'binary' | 'decimal' = 'binary',
): string {
  return `${formatBytes(bytesPerSec, units)}/s`
}

export function formatDuration(seconds: number): string {
  const s = safeNumber(Math.floor(seconds))
  if (s < 60) return `${s}s`
  const m = Math.floor(s / 60)
  const remS = s % 60
  if (m < 60) return remS > 0 ? `${m}m ${remS}s` : `${m}m`
  const h = Math.floor(m / 60)
  const remM = m % 60
  if (h < 24) return remM > 0 ? `${h}h ${remM}m` : `${h}h`
  const d = Math.floor(h / 24)
  const remH = h % 24
  return remH > 0 ? `${d}d ${remH}h` : `${d}d`
}

export function formatNumber(n: number): string {
  return safeNumber(n).toLocaleString('en-US')
}

export function splitRate(
  bytesPerSec: number,
  units: 'binary' | 'decimal' = 'binary',
): { value: string; unit: string } {
  const b = safeNumber(bytesPerSec)
  const base = pickBase(units)
  const labels = pickLabels(units)
  if (b < base) return { value: b.toFixed(0), unit: `${labels[0]}/s` }
  const i = Math.min(
    Math.floor(Math.log(b) / Math.log(base)),
    labels.length - 1,
  )
  const val = b / Math.pow(base, i)
  return { value: val.toFixed(2), unit: `${labels[i]}/s` }
}

export function splitBytes(
  bytes: number,
  units: 'binary' | 'decimal' = 'binary',
): { value: string; unit: string } {
  const b = safeNumber(bytes)
  const base = pickBase(units)
  const labels = pickLabels(units)
  if (b < base) return { value: b.toFixed(0), unit: labels[0] }
  const i = Math.min(
    Math.floor(Math.log(b) / Math.log(base)),
    labels.length - 1,
  )
  const val = b / Math.pow(base, i)
  return { value: val.toFixed(2), unit: labels[i] }
}

export function useFormat() {
  return {
    formatBytes,
    formatRate,
    formatDuration,
    formatNumber,
    splitRate,
    splitBytes,
  }
}
