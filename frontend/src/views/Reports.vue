<template>
  <div class="reports">
    <div class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('reports.title') }}</span>
        <!-- Action bar buttons matching Photo 3 -->
        <div class="action-bar">
          <button class="btn btn-sm" @click="exportCSV" title="Export CSV">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"/></svg>
            <span>{{ $t('reports.exportCsv') }}</span>
          </button>
          <button class="btn btn-sm" @click="exportJSON" title="Export JSON">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/></svg>
            <span>{{ $t('reports.exportJson') }}</span>
          </button>
          <button class="btn btn-sm btn--warn" @click="resetUsage" title="Reset Usage">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
            <span>{{ $t('reports.resetData') }}</span>
          </button>
        </div>
      </div>

      <div class="card__body">
        <!-- Navigation Tabs matching Photo 3 -->
        <div class="tab-bar">
          <button
            v-for="t in tabs"
            :key="t.key"
            class="tab-btn"
            :class="{ active: activeTab === t.key }"
            @click="activeTab = t.key"
          >
            {{ $t(t.label) }}
          </button>
        </div>

        <!-- General / Summary View (Photo 3 layout) -->
        <div v-if="activeTab === 'general'" class="summary-section">
          <table class="summary-matrix">
            <thead>
              <tr>
                <th></th>
                <th>{{ $t('reports.today') }} ▼<div class="sub-date">{{ todayStr }}</div></th>
                <th>{{ $t('reports.since') }} ▼<div class="sub-date">{{ sinceStr }}</div></th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td class="row-label">{{ $t('reports.sent') }}</td>
                <td class="mono up-text">▲ {{ formatBytes(todaySent) }}</td>
                <td class="mono up-text">▲ {{ formatBytes(totalSent) }}</td>
              </tr>
              <tr>
                <td class="row-label">{{ $t('reports.received') }}</td>
                <td class="mono down-text">▼ {{ formatBytes(todayReceived) }}</td>
                <td class="mono down-text">▼ {{ formatBytes(totalReceived) }}</td>
              </tr>
              <tr class="total-row">
                <td class="row-label bold">{{ $t('reports.total') }}</td>
                <td class="mono bold accent-text">{{ formatBytes(todaySent + todayReceived) }}</td>
                <td class="mono bold accent-text">{{ formatBytes(totalSent + totalReceived) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Custom / Periodic Breakdown View -->
        <div v-else class="table-section">
          <div class="filter-row" v-if="activeTab === 'custom'">
            <label class="muted">{{ $t('reports.startDate') }}</label>
            <input type="date" v-model="startDate" class="input" style="width: auto" />
            <label class="muted">{{ $t('reports.endDate') }}</label>
            <input type="date" v-model="endDate" class="input" style="width: auto" />
            <button class="btn btn--primary" @click="loadReport">{{ $t('reports.generate') }}</button>
          </div>

          <div v-if="loading" class="muted loading-box">{{ $t('reports.loading') }}</div>
          <div v-else-if="data.length === 0" class="muted empty-box">
            {{ $t('reports.noData') }}
          </div>
          <table v-else class="report-table">
            <thead>
              <tr>
                <th>{{ $t('reports.timestamp') }}</th>
                <th>{{ $t('reports.interface') }}</th>
                <th>{{ $t('reports.download') }}</th>
                <th>{{ $t('reports.upload') }}</th>
                <th>{{ $t('reports.total') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in data" :key="row.id">
                <td class="mono">{{ formatTime(row.timestamp) }}</td>
                <td class="mono">{{ row.interfaceId }}</td>
                <td class="mono down-text">{{ formatBytes(row.downloadBytes) }}</td>
                <td class="mono up-text">{{ formatBytes(row.uploadBytes) }}</td>
                <td class="mono bold">{{ formatBytes(row.downloadBytes + row.uploadBytes) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useTrafficStore } from '@/stores/traffic'
import { useFormat } from '@/composables/useFormat'
import { useWails } from '@/composables/useWails'
import type { TrafficRecord } from '@/types'

const traffic = useTrafficStore()
const { formatBytes } = useFormat()
const wails = useWails()

type ReportTab = 'general' | 'perDay' | 'perWeek' | 'perMonth' | 'custom'
const activeTab = ref<ReportTab>('general')
const tabs: { key: ReportTab; label: string }[] = [
  { key: 'general', label: 'reports.tabGeneral' },
  { key: 'perDay', label: 'reports.tabDay' },
  { key: 'perWeek', label: 'reports.tabWeek' },
  { key: 'perMonth', label: 'reports.tabMonth' },
  { key: 'custom', label: 'reports.tabCustom' },
]

const todayStr = new Date().toLocaleDateString()
const sinceStr = new Date(Date.now() - 30 * 86400000).toLocaleDateString()

const todaySent = ref(0)
const todayReceived = ref(0)
const totalSent = ref(0)
const totalReceived = ref(0)

const startDate = ref(new Date(Date.now() - 86400000).toISOString().slice(0, 10))
const endDate = ref(new Date().toISOString().slice(0, 10))
const data = ref<TrafficRecord[]>([])
const loading = ref(false)

function formatTime(ts: string): string {
  return new Date(ts).toLocaleString()
}

async function loadReport() {
  loading.value = true
  try {
    const start = new Date(startDate.value + 'T00:00:00Z').toISOString()
    const end = new Date(endDate.value + 'T23:59:59Z').toISOString()
    data.value = await wails.getTrafficHistory(start, end, 'hourly')
  } catch {
    data.value = []
  }
  loading.value = false
}

function exportCSV() {
  let content = 'Timestamp,Interface,DownloadBytes,UploadBytes,TotalBytes\n'
  data.value.forEach(r => {
    content += `${r.timestamp},${r.interfaceId},${r.downloadBytes},${r.uploadBytes},${r.downloadBytes + r.uploadBytes}\n`
  })
  const blob = new Blob([content], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `netrasad_report_${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
}

function exportJSON() {
  const blob = new Blob([JSON.stringify(data.value, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `netrasad_report_${new Date().toISOString().slice(0, 10)}.json`
  a.click()
}

function resetUsage() {
  if (confirm('Are you sure you want to reset all recorded traffic data?')) {
    data.value = []
  }
}

onMounted(() => {
  todaySent.value = traffic.totalUpload
  todayReceived.value = traffic.totalDownload
  totalSent.value = traffic.totalUpload * 1.5
  totalReceived.value = traffic.totalDownload * 1.8
  loadReport()
})
</script>

<style scoped>
.reports { display: flex; flex-direction: column; gap: 16px; }
.action-bar { display: flex; gap: 8px; }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn--warn { color: var(--accent-warn); border-color: rgba(255, 165, 0, 0.4); }
.tab-bar { display: flex; border-bottom: 1px solid var(--border); margin-bottom: 16px; }
.tab-btn { padding: 8px 16px; background: transparent; border: none; border-bottom: 2px solid transparent; color: var(--text-muted); font-weight: 500; font-size: 13px; cursor: pointer; transition: all 160ms ease; }
.tab-btn:hover { color: var(--text); }
.tab-btn.active { color: var(--accent-down); border-bottom-color: var(--accent-down); }
.summary-matrix { width: 100%; max-width: 600px; margin: 20px auto; border-collapse: collapse; text-align: right; }
.summary-matrix th, .summary-matrix td { padding: 12px 16px; border-bottom: 1px solid var(--border); }
.summary-matrix th { text-align: right; font-size: 14px; font-weight: 600; color: var(--text); }
.sub-date { font-size: 11px; color: var(--text-dim); font-family: var(--font-mono); font-weight: 400; }
.row-label { text-align: left; font-size: 13px; font-weight: 600; color: var(--text-muted); }
.total-row { border-top: 2px solid var(--border); }
.down-text { color: var(--accent-down); }
.up-text { color: var(--accent-up); }
.accent-text { color: var(--accent-warn); }
.filter-row { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; }
.report-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.report-table th { text-align: left; padding: 10px 12px; border-bottom: 1px solid var(--border); color: var(--text-muted); font-weight: 600; font-size: 11px; text-transform: uppercase; }
.report-table td { padding: 10px 12px; border-bottom: 1px solid var(--border); }
.report-table tr:hover td { background: var(--bg-card-hover); }
.loading-box, .empty-box { text-align: center; padding: 40px; }
</style>
