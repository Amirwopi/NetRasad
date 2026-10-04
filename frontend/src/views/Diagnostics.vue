<template>
  <div class="diagnostics">
    <!-- Tool selector -->
    <div class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('diagnostics.title') }}</span>
      </div>
      <div class="card__body">
        <div class="tool-tabs">
          <button
            v-for="tool in tools"
            :key="tool.id"
            class="tool-tab"
            :class="{ 'tool-tab--active': activeTool === tool.id }"
            @click="activeTool = tool.id"
          >
            {{ $t(`diagnostics.tools.${tool.id}`) }}
          </button>
        </div>

        <!-- Input form -->
        <div class="tool-form">
          <div class="form-row">
            <label class="form-label">{{ $t('diagnostics.host') }}</label>
            <input
              v-model="host"
              type="text"
              class="form-input"
              :placeholder="$t('diagnostics.hostPlaceholder')"
              @keyup.enter="runTool"
            />
          </div>
          <div v-if="activeTool === 'tcp'" class="form-row form-row--port">
            <label class="form-label">{{ $t('diagnostics.port') }}</label>
            <input
              v-model.number="port"
              type="number"
              class="form-input form-input--port"
              min="1"
              max="65535"
              @keyup.enter="runTool"
            />
          </div>
          <div v-if="activeTool === 'ping'" class="form-row form-row--count">
            <label class="form-label">{{ $t('diagnostics.count') }}</label>
            <input
              v-model.number="count"
              type="number"
              class="form-input form-input--count"
              min="1"
              max="100"
            />
          </div>
          <button
            class="btn btn--primary"
            :disabled="loading || !host"
            @click="runTool"
          >
            {{ loading ? $t('diagnostics.running') : $t('diagnostics.run') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Gateway info card (always visible) -->
    <div class="card" v-if="gateway">
      <div class="card__header">
        <span class="card__title">{{ $t('diagnostics.gatewayTitle') }}</span>
      </div>
      <div class="card__body">
        <div class="gateway-info">
          <div class="info-row">
            <span class="info-label">{{ $t('diagnostics.gatewayIP') }}</span>
            <span class="info-value">{{ gateway.GatewayIP }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">{{ $t('diagnostics.gatewayInterface') }}</span>
            <span class="info-value">{{ gateway.Interface || '—' }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">{{ $t('diagnostics.gatewayMetric') }}</span>
            <span class="info-value">{{ gateway.Metric }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Error -->
    <div v-if="error" class="card">
      <div class="card__body">
        <div class="error-msg">{{ error }}</div>
      </div>
    </div>

    <!-- Ping results -->
    <div v-if="pingResult" class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('diagnostics.pingResult') }} — {{ pingResult.Host }}</span>
      </div>
      <div class="card__body">
        <div class="result-grid">
          <div class="result-stat">
            <span class="result-stat__label">{{ $t('diagnostics.sent') }}</span>
            <span class="result-stat__value">{{ pingResult.Sent }}</span>
          </div>
          <div class="result-stat">
            <span class="result-stat__label">{{ $t('diagnostics.received') }}</span>
            <span class="result-stat__value">{{ pingResult.Received }}</span>
          </div>
          <div class="result-stat">
            <span class="result-stat__label">{{ $t('diagnostics.packetLoss') }}</span>
            <span class="result-stat__value" :class="{ 'text-danger': pingResult.PacketLoss > 0 }">
              {{ pingResult.PacketLoss.toFixed(1) }}%
            </span>
          </div>
          <div class="result-stat">
            <span class="result-stat__label">{{ $t('diagnostics.minRTT') }}</span>
            <span class="result-stat__value">{{ formatRTT(pingResult.MinRTT) }}</span>
          </div>
          <div class="result-stat">
            <span class="result-stat__label">{{ $t('diagnostics.avgRTT') }}</span>
            <span class="result-stat__value">{{ formatRTT(pingResult.AvgRTT) }}</span>
          </div>
          <div class="result-stat">
            <span class="result-stat__label">{{ $t('diagnostics.maxRTT') }}</span>
            <span class="result-stat__value">{{ formatRTT(pingResult.MaxRTT) }}</span>
          </div>
        </div>
        <div class="rtt-bars" v-if="pingResult.RTTs.length">
          <div
            v-for="(rtt, i) in pingResult.RTTs"
            :key="i"
            class="rtt-bar"
            :style="{ height: rttBarHeight(rtt) + '%' }"
            :title="formatRTT(rtt)"
          ></div>
        </div>
        <details class="raw-output">
          <summary>{{ $t('diagnostics.rawOutput') }}</summary>
          <pre>{{ pingResult.Output }}</pre>
        </details>
      </div>
    </div>

    <!-- Traceroute results -->
    <div v-if="tracerouteResult" class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('diagnostics.tracerouteResult') }}</span>
      </div>
      <div class="card__body">
        <table class="data-table" v-if="tracerouteResult.Hops.length">
          <thead>
            <tr>
              <th>{{ $t('diagnostics.hop') }}</th>
              <th>{{ $t('diagnostics.ip') }}</th>
              <th>{{ $t('diagnostics.rtt1') }}</th>
              <th>{{ $t('diagnostics.rtt2') }}</th>
              <th>{{ $t('diagnostics.rtt3') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="hop in tracerouteResult.Hops" :key="hop.Hop">
              <td>{{ hop.Hop }}</td>
              <td>{{ hop.IP || hop.Host || '*' }}</td>
              <td>{{ formatRTT(hop.RTT1) }}</td>
              <td>{{ formatRTT(hop.RTT2) }}</td>
              <td>{{ formatRTT(hop.RTT3) }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="muted">{{ $t('diagnostics.noHops') }}</p>
        <details class="raw-output">
          <summary>{{ $t('diagnostics.rawOutput') }}</summary>
          <pre>{{ tracerouteResult.Output }}</pre>
        </details>
      </div>
    </div>

    <!-- DNS results -->
    <div v-if="dnsResult" class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('diagnostics.dnsResult') }} — {{ dnsResult.Host }}</span>
      </div>
      <div class="card__body">
        <div v-if="dnsResult.CNAME" class="info-row">
          <span class="info-label">CNAME</span>
          <span class="info-value">{{ dnsResult.CNAME }}</span>
        </div>
        <div class="dns-section" v-if="dnsResult.ARecords.length">
          <h4>A (IPv4)</h4>
          <ul class="dns-list">
            <li v-for="r in dnsResult.ARecords" :key="r">{{ r }}</li>
          </ul>
        </div>
        <div class="dns-section" v-if="dnsResult.AAAARecords.length">
          <h4>AAAA (IPv6)</h4>
          <ul class="dns-list">
            <li v-for="r in dnsResult.AAAARecords" :key="r">{{ r }}</li>
          </ul>
        </div>
        <div class="dns-section" v-if="dnsResult.NSRecords.length">
          <h4>NS</h4>
          <ul class="dns-list">
            <li v-for="r in dnsResult.NSRecords" :key="r">{{ r }}</li>
          </ul>
        </div>
        <div class="dns-section" v-if="dnsResult.MXRecords.length">
          <h4>MX</h4>
          <ul class="dns-list">
            <li v-for="r in dnsResult.MXRecords" :key="r">{{ r }}</li>
          </ul>
        </div>
        <p v-if="!dnsResult.ARecords.length && !dnsResult.AAAARecords.length && !dnsResult.NSRecords.length && !dnsResult.MXRecords.length && !dnsResult.CNAME" class="muted">
          {{ $t('diagnostics.noRecords') }}
        </p>
      </div>
    </div>

    <!-- TCP results -->
    <div v-if="tcpResult" class="card">
      <div class="card__header">
        <span class="card__title">{{ $t('diagnostics.tcpResult') }} — {{ tcpResult.Host }}:{{ tcpResult.Port }}</span>
      </div>
      <div class="card__body">
        <div class="result-grid">
          <div class="result-stat">
            <span class="result-stat__label">{{ $t('diagnostics.status') }}</span>
            <span class="result-stat__value" :class="tcpResult.Success ? 'text-success' : 'text-danger'">
              {{ tcpResult.Success ? $t('diagnostics.connected') : $t('diagnostics.failed') }}
            </span>
          </div>
          <div class="result-stat">
            <span class="result-stat__label">{{ $t('diagnostics.latency') }}</span>
            <span class="result-stat__value">{{ formatRTT(tcpResult.RTT) }}</span>
          </div>
        </div>
        <div v-if="tcpResult.Error" class="error-msg">{{ tcpResult.Error }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useWails } from '@/composables/useWails'
import type { PingResult, TracerouteResult, DNSResult, TCPResult, GatewayInfo } from '@/types'

const {
  isWails,
  ping: pingFn,
  traceroute: tracerouteFn,
  dnsLookup,
  tcpConnect,
  getDefaultGateway,
} = useWails()

const tools = [
  { id: 'ping' },
  { id: 'traceroute' },
  { id: 'dns' },
  { id: 'tcp' },
] as const

const activeTool = ref<'ping' | 'traceroute' | 'dns' | 'tcp'>('ping')
const host = ref('')
const port = ref(443)
const count = ref(4)
const loading = ref(false)
const error = ref('')

const pingResult = ref<PingResult | null>(null)
const tracerouteResult = ref<TracerouteResult | null>(null)
const dnsResult = ref<DNSResult | null>(null)
const tcpResult = ref<TCPResult | null>(null)
const gateway = ref<GatewayInfo | null>(null)

onMounted(async () => {
  if (!isWails.value) return
  try {
    gateway.value = await getDefaultGateway()
  } catch (e) {
  }
})

function clearResults() {
  pingResult.value = null
  tracerouteResult.value = null
  dnsResult.value = null
  tcpResult.value = null
  error.value = ''
}

async function runTool() {
  if (!host.value || loading.value) return
  loading.value = true
  clearResults()
  try {
    switch (activeTool.value) {
      case 'ping':
        pingResult.value = await pingFn(host.value, count.value)
        break
      case 'traceroute':
        tracerouteResult.value = await tracerouteFn(host.value, 30)
        break
      case 'dns':
        dnsResult.value = await dnsLookup(host.value)
        break
      case 'tcp':
        tcpResult.value = await tcpConnect(host.value, port.value, 5)
        break
    }
  } catch (e: any) {
    error.value = String(e?.message || e)
  } finally {
    loading.value = false
  }
}

function formatRTT(ms: number): string {
  if (ms < 0) return '*'
  if (ms < 1) return `${(ms * 1000).toFixed(0)}μs`
  return `${ms.toFixed(2)}ms`
}

function rttBarHeight(rtt: number): number {
  if (rtt < 0) return 5
  const max = pingResult.value?.MaxRTT || 1
  return Math.max(5, Math.min(100, (rtt / max) * 100))
}
</script>

<style scoped>
.diagnostics { display: flex; flex-direction: column; gap: 16px; }

.tool-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.tool-tab {
  padding: 8px 16px;
  border: 1px solid var(--border, #333);
  background: transparent;
  color: var(--text-muted, #888);
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.15s;
}
.tool-tab:hover { color: var(--text); border-color: var(--accent-down); }
.tool-tab--active {
  background: var(--accent-down);
  color: #fff;
  border-color: var(--accent-down);
}

.tool-form {
  display: flex;
  gap: 12px;
  align-items: flex-end;
  flex-wrap: wrap;
}
.form-row { display: flex; flex-direction: column; gap: 4px; }
.form-row--port, .form-row--count { min-width: 90px; }
.form-label { font-size: 12px; color: var(--text-muted); }
.form-input {
  padding: 8px 12px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text);
  font-size: 14px;
  min-width: 220px;
}
.form-input--port, .form-input--count { min-width: 80px; }
.form-input:focus { outline: none; border-color: var(--accent-down); }

.btn {
  padding: 8px 20px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
}
.btn--primary {
  background: var(--accent-down);
  color: #fff;
}
.btn--primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn--primary:hover:not(:disabled) { filter: brightness(1.1); }

.result-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}
.result-stat { display: flex; flex-direction: column; gap: 2px; }
.result-stat__label { font-size: 11px; color: var(--text-muted); text-transform: uppercase; }
.result-stat__value { font-size: 18px; font-weight: 600; }

.rtt-bars {
  display: flex;
  gap: 4px;
  align-items: flex-end;
  height: 60px;
  margin-bottom: 16px;
}
.rtt-bar {
  flex: 1;
  background: var(--accent-down);
  border-radius: 3px 3px 0 0;
  min-width: 8px;
  transition: height 0.3s;
}

.gateway-info, .info-row { display: flex; gap: 12px; }
.info-row { margin-bottom: 6px; }
.info-label { font-size: 12px; color: var(--text-muted); min-width: 100px; }
.info-value { font-size: 14px; }

.data-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.data-table th, .data-table td {
  padding: 8px 12px;
  text-align: left;
  border-bottom: 1px solid var(--border);
}
.data-table th { color: var(--text-muted); font-weight: 600; font-size: 11px; text-transform: uppercase; }

.dns-section { margin-bottom: 12px; }
.dns-section h4 { font-size: 13px; margin-bottom: 4px; color: var(--accent-down); }
.dns-list { list-style: none; padding: 0; margin: 0; }
.dns-list li { padding: 4px 0; font-size: 13px; font-family: monospace; }

.raw-output { margin-top: 12px; }
.raw-output summary { cursor: pointer; font-size: 12px; color: var(--text-muted); }
.raw-output pre {
  margin-top: 8px;
  padding: 12px;
  background: var(--bg-input);
  border-radius: 6px;
  font-size: 12px;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

.error-msg { color: var(--accent-error); font-size: 13px; }
.text-success { color: var(--accent-up); }
.text-danger { color: var(--accent-error); }
.muted { color: var(--text-muted); font-size: 13px; }
</style>
