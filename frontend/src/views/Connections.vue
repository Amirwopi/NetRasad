<template>
  <div class="connections">
    <!-- Toolbar -->
    <div class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('connections.title') }}</span>
        <div class="toolbar">
          <div class="search-box">
            <input
              v-model="searchQuery"
              type="text"
              class="form-input"
              :placeholder="$t('connections.searchPlaceholder')"
            />
          </div>
          <select v-model="protocolFilter" class="form-input form-input--select">
            <option value="all">{{ $t('connections.allProtocols') }}</option>
            <option value="tcp">TCP</option>
            <option value="udp">UDP</option>
          </select>
          <select v-model="stateFilter" class="form-input form-input--select">
            <option value="all">{{ $t('connections.allStates') }}</option>
            <option v-for="s in states" :key="s" :value="s">{{ s }}</option>
          </select>
          <button class="btn btn--primary" :disabled="loading" @click="refresh">
            {{ loading ? $t('connections.refreshing') : $t('connections.refresh') }}
          </button>
        </div>
      </div>
      <div class="card__body">
        <!-- Summary stats -->
        <div class="summary-grid" v-if="connections.length">
          <div class="summary-stat">
            <span class="summary-stat__label">{{ $t('connections.total') }}</span>
            <span class="summary-stat__value">{{ connections.length }}</span>
          </div>
          <div class="summary-stat">
            <span class="summary-stat__label">{{ $t('connections.tcpCount') }}</span>
            <span class="summary-stat__value">{{ tcpCount }}</span>
          </div>
          <div class="summary-stat">
            <span class="summary-stat__label">{{ $t('connections.udpCount') }}</span>
            <span class="summary-stat__value">{{ udpCount }}</span>
          </div>
          <div class="summary-stat">
            <span class="summary-stat__label">{{ $t('connections.established') }}</span>
            <span class="summary-stat__value text-success">{{ establishedCount }}</span>
          </div>
        </div>

        <!-- Error -->
        <div v-if="error" class="error-msg">{{ error }}</div>

        <!-- Empty state -->
        <div v-if="!loading && !filteredConnections.length && !error" class="empty-state">
          <p class="muted">{{ $t('connections.noConnections') }}</p>
        </div>

        <!-- Connections table -->
        <table class="conn-table" v-if="filteredConnections.length">
          <thead>
            <tr>
              <th @click="sortBy('Protocol')" class="sortable">
                {{ $t('connections.protocol') }}
                <span v-if="sortKey === 'Protocol'" class="sort-arrow">{{ sortOrder > 0 ? '▲' : '▼' }}</span>
              </th>
              <th @click="sortBy('LocalAddress')" class="sortable">
                {{ $t('connections.localAddress') }}
                <span v-if="sortKey === 'LocalAddress'" class="sort-arrow">{{ sortOrder > 0 ? '▲' : '▼' }}</span>
              </th>
              <th @click="sortBy('RemoteAddress')" class="sortable">
                {{ $t('connections.remoteAddress') }}
                <span v-if="sortKey === 'RemoteAddress'" class="sort-arrow">{{ sortOrder > 0 ? '▲' : '▼' }}</span>
              </th>
              <th @click="sortBy('State')" class="sortable">
                {{ $t('connections.state') }}
                <span v-if="sortKey === 'State'" class="sort-arrow">{{ sortOrder > 0 ? '▲' : '▼' }}</span>
              </th>
              <th @click="sortBy('PID')" class="sortable">
                {{ $t('connections.pid') }}
                <span v-if="sortKey === 'PID'" class="sort-arrow">{{ sortOrder > 0 ? '▲' : '▼' }}</span>
              </th>
              <th @click="sortBy('ProcessName')" class="sortable">
                {{ $t('connections.process') }}
                <span v-if="sortKey === 'ProcessName'" class="sort-arrow">{{ sortOrder > 0 ? '▲' : '▼' }}</span>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(conn, i) in paginatedConnections" :key="i">
              <td>
                <span class="proto-badge" :class="`proto-badge--${conn.Protocol}`">{{ conn.Protocol.toUpperCase() }}</span>
              </td>
              <td class="mono">{{ conn.LocalAddress }}:{{ conn.LocalPort }}</td>
              <td class="mono">{{ conn.RemoteAddress }}:{{ conn.RemotePort }}</td>
              <td>
                <span class="state-badge" :class="stateClass(conn.State)">{{ conn.State }}</span>
              </td>
              <td class="mono">{{ conn.PID || '—' }}</td>
              <td>{{ conn.ProcessName || '—' }}</td>
            </tr>
          </tbody>
        </table>

        <!-- Pagination -->
        <div class="pagination" v-if="filteredConnections.length > pageSize">
          <button
            class="btn btn--small"
            :disabled="currentPage === 0"
            @click="currentPage--"
          >
            {{ $t('connections.prev') }}
          </button>
          <span class="page-info">
            {{ currentPage * pageSize + 1 }}–{{ Math.min((currentPage + 1) * pageSize, filteredConnections.length) }}
            / {{ filteredConnections.length }}
          </span>
          <button
            class="btn btn--small"
            :disabled="(currentPage + 1) * pageSize >= filteredConnections.length"
            @click="currentPage++"
          >
            {{ $t('connections.next') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useWails } from '@/composables/useWails'
import type { Connection, TCPState } from '@/types'

const { isWails, getConnections } = useWails()

const connections = ref<Connection[]>([])
const loading = ref(false)
const error = ref('')
const searchQuery = ref('')
const protocolFilter = ref<'all' | 'tcp' | 'udp'>('all')
const stateFilter = ref<'all' | TCPState>('all')
const sortKey = ref<keyof Connection | ''>('')
const sortOrder = ref(1)
const currentPage = ref(0)
const pageSize = 50
let pollTimer: ReturnType<typeof setInterval> | null = null

const states: TCPState[] = [
  'ESTABLISHED', 'LISTEN', 'TIME_WAIT', 'CLOSE_WAIT',
  'SYN_SENT', 'SYN_RECEIVED', 'FIN_WAIT1', 'FIN_WAIT2',
  'CLOSING', 'LAST_ACK', 'CLOSED',
]

const filteredConnections = computed(() => {
  let list = connections.value

  if (protocolFilter.value !== 'all') {
    list = list.filter(c => c.Protocol === protocolFilter.value)
  }

  if (stateFilter.value !== 'all') {
    list = list.filter(c => c.State === stateFilter.value)
  }

  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(c =>
      c.LocalAddress.toLowerCase().includes(q) ||
      c.RemoteAddress.toLowerCase().includes(q) ||
      c.ProcessName.toLowerCase().includes(q) ||
      String(c.PID).includes(q) ||
      String(c.LocalPort).includes(q) ||
      String(c.RemotePort).includes(q)
    )
  }

  if (sortKey.value) {
    const key = sortKey.value
    list = [...list].sort((a, b) => {
      const av = String(a[key] ?? '')
      const bv = String(b[key] ?? '')
      if (av < bv) return -1 * sortOrder.value
      if (av > bv) return 1 * sortOrder.value
      return 0
    })
  }

  return list
})

const paginatedConnections = computed(() => {
  const start = currentPage.value * pageSize
  return filteredConnections.value.slice(start, start + pageSize)
})

const tcpCount = computed(() => connections.value.filter(c => c.Protocol === 'tcp').length)
const udpCount = computed(() => connections.value.filter(c => c.Protocol === 'udp').length)
const establishedCount = computed(() => connections.value.filter(c => c.State === 'ESTABLISHED').length)

function sortBy(key: keyof Connection) {
  if (sortKey.value === key) {
    sortOrder.value *= -1
  } else {
    sortKey.value = key
    sortOrder.value = 1
  }
}

function stateClass(state: TCPState): string {
  switch (state) {
    case 'ESTABLISHED': return 'state-badge--established'
    case 'LISTEN': return 'state-badge--listen'
    case 'TIME_WAIT': return 'state-badge--timewait'
    case 'CLOSE_WAIT': return 'state-badge--closewait'
    default: return 'state-badge--other'
  }
}

async function refresh() {
  if (!isWails.value || loading.value) return
  loading.value = true
  error.value = ''
  try {
    connections.value = await getConnections()
    currentPage.value = 0
  } catch (e: any) {
    error.value = String(e?.message || e)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refresh()
  pollTimer = setInterval(refresh, 5000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped>
.connections { display: flex; flex-direction: column; gap: 16px; }

.toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.search-box { flex: 1; min-width: 200px; }
.form-input {
  padding: 6px 12px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text);
  font-size: 13px;
  width: 100%;
}
.form-input--select { width: auto; min-width: 120px; cursor: pointer; }
.form-input:focus { outline: none; border-color: var(--accent-down); }

.btn {
  padding: 6px 16px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
}
.btn--small { padding: 4px 12px; font-size: 12px; }
.btn--primary {
  background: var(--accent, #00D4FF);
  color: #000;
}
.btn--primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn--primary:hover:not(:disabled) { filter: brightness(1.1); }

.summary-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}
.summary-stat { display: flex; flex-direction: column; gap: 2px; }
.summary-stat__label { font-size: 11px; color: var(--text-muted, #888); text-transform: uppercase; }
.summary-stat__value { font-size: 20px; font-weight: 700; }

.conn-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.conn-table th, .conn-table td {
  padding: 6px 10px;
  text-align: left;
  border-bottom: 1px solid var(--border, #333);
}
.conn-table th {
  color: var(--text-muted, #888);
  font-weight: 600;
  font-size: 11px;
  text-transform: uppercase;
  white-space: nowrap;
}
.sortable { cursor: pointer; user-select: none; }
.sortable:hover { color: var(--accent, #00D4FF); }
.sort-arrow { font-size: 10px; margin-left: 2px; }
.mono { font-family: 'Cascadia Code', 'Consolas', monospace; }

.proto-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.5px;
}
.proto-badge--tcp { background: rgba(0, 212, 255, 0.15); color: #00D4FF; }
.proto-badge--udp { background: rgba(0, 255, 136, 0.15); color: #00FF88; }

.state-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
}
.state-badge--established { background: rgba(0, 255, 136, 0.15); color: #00FF88; }
.state-badge--listen { background: rgba(0, 212, 255, 0.15); color: #00D4FF; }
.state-badge--timewait { background: rgba(255, 165, 0, 0.15); color: #FFA500; }
.state-badge--closewait { background: rgba(255, 107, 107, 0.15); color: #ff6b6b; }
.state-badge--other { background: rgba(128, 128, 128, 0.15); color: #888; }

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-top: 12px;
}
.page-info { font-size: 12px; color: var(--text-muted, #888); }

.empty-state { text-align: center; padding: 24px; }
.error-msg { color: #ff6b6b; font-size: 13px; margin-bottom: 12px; }
.text-success { color: #00FF88; }
.muted { color: var(--text-muted, #888); font-size: 13px; }
</style>
