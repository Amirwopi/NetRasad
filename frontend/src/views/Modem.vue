<template>
  <div class="modem-container fade-in">
    <div class="header">
      <h2>{{ $t('modem.title', 'Modem Settings') }}</h2>
      <p class="subtitle">{{ $t('modem.subtitle', 'Configure and monitor your modem/router.') }}</p>
    </div>

    <!-- Connection Card -->
    <div class="settings-card">
      <h3 class="card-title">{{ $t('modem.connection', 'Connection Settings') }}</h3>
      
      <div class="row">
        <div class="setting-group flex-1">
          <label>{{ $t('modem.host', 'Modem IP') }}</label>
          <input type="text" v-model="host" placeholder="192.168.1.1" class="text-input" />
        </div>
        <div class="setting-group flex-1">
          <label>{{ $t('modem.brand', 'Modem Brand') }}</label>
          <select v-model="brand" class="text-input">
            <option value="tp-link">TP-Link / Broadcom</option>
            <option value="d-link">D-Link / Ralink</option>
            <option value="generic">Generic (Basic)</option>
          </select>
        </div>
      </div>

      <div class="row">
        <div class="setting-group flex-1">
          <label>{{ $t('modem.username', 'Username') }}</label>
          <input type="text" v-model="username" placeholder="admin" class="text-input" />
        </div>
        <div class="setting-group flex-1">
          <label>{{ $t('modem.password', 'Password') }}</label>
          <input type="password" v-model="password" placeholder="admin" class="text-input" />
        </div>
      </div>

      <div class="btn-group">
        <button @click="testConnection" class="primary-btn" :disabled="loading">
          <span v-if="loading && action === 'test'" class="spinner"></span>
          <span v-else>{{ $t('modem.testConn', 'Test Connection') }}</span>
        </button>
        <button @click="rebootModem" class="pill-btn danger-btn" :disabled="loading">
          <span v-if="loading && action === 'reboot'" class="spinner spinner-red"></span>
          <span v-else>{{ $t('modem.reboot', 'Reboot Modem') }}</span>
        </button>
      </div>

      <div v-if="dnsServers.length > 0" class="info-row" style="margin-top: 16px; display: flex; gap: 8px; align-items: center;">
        <span style="font-size: 12px; color: var(--text-muted);">Current DNS:</span>
        <span class="mono" style="font-size: 13px;">{{ dnsServers.join(', ') }}</span>
      </div>
      
      <div v-if="statusMessage" :class="['status-box', statusError ? 'error' : 'success']">
        {{ statusMessage }}
      </div>
    </div>

    <!-- Tabbed Dashboard -->
    <div class="settings-card" v-if="adslStatus || devices.length > 0 || fwLoading">
      <div class="tabs-container">
        <button class="tab-btn" :class="{active: activeTab === 'adsl'}" @click="activeTab = 'adsl'">
          <span class="icon">📈</span> ADSL Status
        </button>
        <button class="tab-btn" :class="{active: activeTab === 'devices'}" @click="activeTab = 'devices'">
          <span class="icon">💻</span> Devices ({{ devices.length }})
        </button>
        <button class="tab-btn" :class="{active: activeTab === 'firewall'}" @click="activeTab = 'firewall'">
          <span class="icon">🛡️</span> Firewall
        </button>
        <button class="tab-btn" :class="{active: activeTab === 'nat'}" @click="activeTab = 'nat'">
          <span class="icon">🔄</span> NAT / UPnP
        </button>
        <button class="tab-btn" :class="{active: activeTab === 'routes'}" @click="activeTab = 'routes'">
          <span class="icon">🛣️</span> Routing
        </button>
        <button class="tab-btn" :class="{active: activeTab === 'interfaces'}" @click="activeTab = 'interfaces'">
          <span class="icon">🌐</span> Interfaces
        </button>
      </div>

      <!-- ADSL Tab -->
      <div v-if="activeTab === 'adsl'" class="tab-content">
        <div class="card-header">
          <h3 class="card-title">ADSL Line Status</h3>
          <div style="display: flex; gap: 8px;">
            <button @click="toggleLiveScan" class="pill-btn" :class="{'active-scan': isLiveScan}" style="transition: all 0.3s ease;">
              <span v-if="isLiveScan" class="spinner spinner-dark" style="margin-right: 4px;"></span>
              {{ isLiveScan ? 'Stop Scan' : 'Live Scan (5s)' }}
            </button>
            <button @click="fetchAdslStatus" class="pill-btn" :disabled="adslLoading || isLiveScan">
              <span v-if="adslLoading && !isLiveScan" class="spinner spinner-dark"></span>
              <span v-else>Refresh Line</span>
            </button>
          </div>
        </div>

        <div class="adsl-grid" v-if="adslStatus">
          <div class="adsl-box">
            <div class="adsl-icon down-icon">↓</div>
            <div class="adsl-info">
              <span class="adsl-label">Downstream Rate</span>
              <span class="adsl-value">{{ adslStatus.downstreamRate || 'N/A' }} <small>Kbps</small></span>
            </div>
          </div>
          
          <div class="adsl-box">
            <div class="adsl-icon up-icon">↑</div>
            <div class="adsl-info">
              <span class="adsl-label">Upstream Rate</span>
              <span class="adsl-value">{{ adslStatus.upstreamRate || 'N/A' }} <small>Kbps</small></span>
            </div>
          </div>

          <div class="adsl-box">
            <div class="adsl-icon snr-icon">〰</div>
            <div class="adsl-info">
              <span class="adsl-label">Downstream SNR</span>
              <span class="adsl-value">{{ adslStatus.downstreamSNR || 'N/A' }} <small>dB</small></span>
            </div>
          </div>

          <div class="adsl-box">
            <div class="adsl-icon att-icon">⚡</div>
            <div class="adsl-info">
              <span class="adsl-label">Attenuation (Down)</span>
              <span class="adsl-value">{{ adslStatus.downstreamAtt || 'N/A' }} <small>dB</small></span>
            </div>
          </div>
        </div>
        
        <div v-if="adslStatus" style="margin-top: 16px;">
          <details>
            <summary class="mono text-muted" style="cursor:pointer; font-size: 13px;">View Raw ADSL Output</summary>
            <pre class="terminal-output">{{ adslStatus.rawOutput }}</pre>
          </details>
        </div>
      </div>

      <!-- Devices Tab -->
      <div v-if="activeTab === 'devices'" class="tab-content">
        <div class="card-header">
          <h3 class="card-title">Connected Devices</h3>
          <button @click="fetchDevices" class="pill-btn" :disabled="loading && action === 'devices'">
            <span v-if="loading && action === 'devices'" class="spinner spinner-dark"></span>
            <span v-else>Refresh</span>
          </button>
        </div>
        
        <div class="table-container" v-if="devices.length > 0">
          <table class="data-table">
            <thead>
              <tr>
                <th>MAC Address</th>
                <th>IP Address</th>
                <th>Name / Status</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="dev in devices" :key="dev.mac">
                <td class="mono text-primary">{{ dev.mac }}</td>
                <td class="mono">{{ dev.ip }}</td>
                <td>
                  <span class="badge success">Active</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty-state">No devices found.</div>
      </div>

      <!-- Firewall Tab -->
      <div v-if="activeTab === 'firewall'" class="tab-content">
        <div class="card-header">
          <h3 class="card-title">Firewall Rules (iptables)</h3>
          <button @click="fetchFirewall" class="pill-btn" :disabled="fwLoading">
            <span v-if="fwLoading" class="spinner spinner-dark"></span>
            <span v-else>Refresh</span>
          </button>
        </div>
        <div class="table-container" v-if="firewallRules.length > 0">
          <table class="data-table compact">
            <thead>
              <tr>
                <th>Chain</th>
                <th>Target</th>
                <th>Protocol</th>
                <th>Source</th>
                <th>Destination</th>
                <th>Extra</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(rule, idx) in firewallRules" :key="idx">
                <td><span class="badge" :class="rule.chain === 'INPUT' ? 'success' : 'primary'">{{ rule.chain }}</span></td>
                <td><span class="badge" :class="rule.target === 'ACCEPT' ? 'success' : (rule.target === 'DROP' ? 'danger' : 'neutral')">{{ rule.target }}</span></td>
                <td class="mono">{{ rule.protocol }}</td>
                <td class="mono">{{ rule.source }}</td>
                <td class="mono">{{ rule.destination }}</td>
                <td class="mono text-muted">{{ rule.extra }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty-state">No firewall rules extracted.</div>
      </div>
      <!-- Routes Tab -->
      <div v-if="activeTab === 'routes'" class="tab-content">
        <div class="card-header">
          <h3 class="card-title">Routing Table</h3>
          <button @click="fetchRoutes" class="pill-btn" :disabled="netLoading">
            <span v-if="netLoading" class="spinner spinner-dark"></span>
            <span v-else>Refresh</span>
          </button>
        </div>
        <div class="table-container" v-if="routes.length > 0">
          <table class="data-table compact">
            <thead>
              <tr>
                <th>Destination</th>
                <th>Gateway</th>
                <th>Genmask</th>
                <th>Flags</th>
                <th>Interface</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(route, idx) in routes" :key="idx">
                <td class="mono text-primary">{{ route.destination }}</td>
                <td class="mono">{{ route.gateway }}</td>
                <td class="mono text-muted">{{ route.genmask }}</td>
                <td><span class="badge neutral">{{ route.flags }}</span></td>
                <td><span class="badge primary">{{ route.iface }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty-state">No routes found.</div>
      </div>

      <!-- Interfaces Tab -->
      <div v-if="activeTab === 'interfaces'" class="tab-content">
        <div class="card-header">
          <h3 class="card-title">Network Interfaces (RX/TX)</h3>
          <button @click="fetchInterfaces" class="pill-btn" :disabled="netLoading">
            <span v-if="netLoading" class="spinner spinner-dark"></span>
            <span v-else>Refresh</span>
          </button>
        </div>
        <div class="table-container" v-if="interfaces.length > 0">
          <table class="data-table compact">
            <thead>
              <tr>
                <th>Interface</th>
                <th>RX Packets</th>
                <th>RX Bytes</th>
                <th>TX Packets</th>
                <th>TX Bytes</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="iface in interfaces" :key="iface.name">
                <td><span class="badge primary">{{ iface.name }}</span></td>
                <td class="mono">{{ iface.rxPackets }}</td>
                <td class="mono text-success">{{ (parseInt(iface.rxBytes) / 1024 / 1024).toFixed(2) }} MB</td>
                <td class="mono">{{ iface.txPackets }}</td>
                <td class="mono text-success">{{ (parseInt(iface.txBytes) / 1024 / 1024).toFixed(2) }} MB</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty-state">No interfaces found.</div>
      </div>

      <!-- NAT Tab -->
      <div v-if="activeTab === 'nat'" class="tab-content">
        <div class="card-header">
          <h3 class="card-title">NAT & UPnP (Port Forwarding)</h3>
          <button @click="fetchNat" class="pill-btn" :disabled="fwLoading">
            <span v-if="fwLoading" class="spinner spinner-dark"></span>
            <span v-else>Refresh</span>
          </button>
        </div>
        <div class="table-container" v-if="natRules.length > 0">
          <table class="data-table compact">
            <thead>
              <tr>
                <th>Chain</th>
                <th>Target</th>
                <th>Protocol</th>
                <th>Source</th>
                <th>Destination</th>
                <th>Extra</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(rule, idx) in natRules" :key="idx">
                <td><span class="badge primary">{{ rule.chain }}</span></td>
                <td><span class="badge success">{{ rule.target }}</span></td>
                <td class="mono">{{ rule.protocol }}</td>
                <td class="mono">{{ rule.source }}</td>
                <td class="mono">{{ rule.destination }}</td>
                <td class="mono text-muted">{{ rule.extra }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty-state">No NAT rules extracted.</div>
      </div>
    </div>

    <!-- WiFi Config Card -->
    <div class="settings-card">
      <h3 class="card-title">{{ $t('modem.wifiConfig', 'WiFi Configuration') }}</h3>
      <div v-if="wifiStatus.length > 0" style="margin-bottom: 16px; display: flex; gap: 8px; flex-wrap: wrap;">
        <span class="badge primary" v-for="(w, i) in wifiStatus" :key="i">
          {{ w.ssid }} <span v-if="w.hidden" style="opacity: 0.7; font-size: 10px;">(Ghost)</span>
        </span>
      </div>
      <div class="row">
        <div class="setting-group flex-1">
          <label>{{ $t('modem.newSsid', 'New WiFi Name (SSID)') }}</label>
          <input type="text" v-model="newSsid" placeholder="MyWiFi" class="text-input" />
        </div>
        <div class="setting-group flex-1">
          <label>{{ $t('modem.newWifiPass', 'New WiFi Password') }}</label>
          <input type="password" v-model="newWifiPass" placeholder="Leave empty to keep" class="text-input" />
        </div>
      </div>
      <div style="display: flex; gap: 12px; margin-top: 16px;">
        <button @click="changeWifi" class="primary-btn" :disabled="loading">
          <span v-if="loading && action === 'wifi'" class="spinner"></span>
          <span v-else>{{ $t('modem.applyWifi', 'Apply WiFi Changes') }}</span>
        </button>
        <button @click="rebootModem" class="pill-btn danger-btn" :disabled="loading" style="display: flex; align-items: center; justify-content: center;">
          <span v-if="loading && action === 'reboot'" class="spinner spinner-red"></span>
          <span v-else>Reboot Modem</span>
        </button>
      </div>
    </div>

    <!-- Raw Command -->
    <div class="settings-card">
      <h3 class="card-title">Raw Telnet Command</h3>
      <div class="command-input-row">
        <input type="text" v-model="command" placeholder="sh, sysinfo, arp show..." class="text-input flex-1" />
        <button @click="runCommand" class="primary-btn" :disabled="loading">
          <span v-if="loading && action === 'cmd'" class="spinner"></span>
          <span v-else>Run</span>
        </button>
      </div>
      <pre v-if="output" class="terminal-output">{{ output }}</pre>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, onBeforeUnmount, onMounted } from 'vue'
import { 
  RunModemCommand, 
  TestModemConnection, 
  GetConnectedDevices, 
  ChangeWifiConfig, 
  RebootModem,
  GetADSLStatus,
  GetFirewallRules,
  GetNATRules,
  GetModemRoutes,
  GetModemInterfaces,
  GetWifiStatus,
  GetDNS
} from '../../wailsjs/go/app/App'

const host = ref('192.168.1.1')
const username = ref('admin')
const password = ref('admin')
const brand = ref('d-link') // Default to D-Link DSL-2877AL

const newSsid = ref('')
const newWifiPass = ref('')
const command = ref('')

const loading = ref(false)
const action = ref('')
const statusMessage = ref('')
const statusError = ref(false)
const output = ref('')
const devices = ref<any[]>([])
const adslStatus = ref<any>(null)
const adslLoading = ref(false)

const firewallRules = ref<any[]>([])
const natRules = ref<any[]>([])
const fwLoading = ref(false)

const routes = ref<any[]>([])
const interfaces = ref<any[]>([])
const netLoading = ref(false)

const wifiStatus = ref<any[]>([])
const dnsServers = ref<string[]>([])

const activeTab = ref('adsl') // 'adsl', 'devices', 'firewall', 'nat', 'routes', 'interfaces'

const isLiveScan = ref(false)
let scanInterval: any = null

const toggleLiveScan = () => {
  if (isLiveScan.value) {
    stopLiveScan()
  } else {
    startLiveScan()
  }
}

const startLiveScan = () => {
  isLiveScan.value = true
  fetchAdslStatus() // fetch immediately
  scanInterval = setInterval(() => {
    if (activeTab.value === 'adsl' && !document.hidden) {
      fetchAdslStatus()
    } else {
      stopLiveScan()
    }
  }, 5000)
}

const stopLiveScan = () => {
  isLiveScan.value = false
  if (scanInterval) clearInterval(scanInterval)
}

const handleVisibilityChange = () => {
  if (document.hidden && isLiveScan.value) {
    stopLiveScan()
  }
}

onMounted(() => {
  document.addEventListener("visibilitychange", handleVisibilityChange)
})

onBeforeUnmount(() => {
  stopLiveScan()
  document.removeEventListener("visibilitychange", handleVisibilityChange)
})

const setStatus = (msg: string, isErr: boolean) => {
  statusMessage.value = msg
  statusError.value = isErr
}

const testConnection = async () => {
  loading.value = true
  action.value = 'test'
  setStatus('', false)
  try {
    await TestModemConnection(host.value, username.value, password.value)
    setStatus('Connection successful! Fetching data...', false)
    await fetchAdslStatus()
    await fetchDevices()
    await fetchFirewall()
    await fetchNat()
    await fetchRoutes()
    await fetchInterfaces()
    await fetchWifiAndDns()
    setStatus('Connection successful! All data loaded.', false)
  } catch (e: any) {
    setStatus(e.toString(), true)
  }
  loading.value = false
}

const fetchFirewall = async () => {
  fwLoading.value = true
  try {
    const res = await GetFirewallRules(host.value, username.value, password.value, brand.value)
    firewallRules.value = res || []
  } catch (e) {
    console.error('Firewall fetch error:', e)
  }
  fwLoading.value = false
}

const fetchWifiAndDns = async () => {
  try {
    const wifi = await GetWifiStatus(host.value, username.value, password.value, brand.value)
    if (wifi) wifiStatus.value = wifi
  } catch (e) { console.error(e) }
  try {
    const dns = await GetDNS(host.value, username.value, password.value, brand.value)
    if (dns) dnsServers.value = dns
  } catch (e) { console.error(e) }
}

const fetchNat = async () => {
  fwLoading.value = true
  try {
    const res = await GetNATRules(host.value, username.value, password.value, brand.value)
    natRules.value = res || []
  } catch (e) {
    console.error('NAT fetch error:', e)
  }
  fwLoading.value = false
}

const fetchRoutes = async () => {
  netLoading.value = true
  try {
    const res = await GetModemRoutes(host.value, username.value, password.value, brand.value)
    routes.value = res || []
  } catch (e) {
    console.error('Routes fetch error:', e)
  }
  netLoading.value = false
}

const fetchInterfaces = async () => {
  netLoading.value = true
  try {
    const res = await GetModemInterfaces(host.value, username.value, password.value, brand.value)
    interfaces.value = res || []
  } catch (e) {
    console.error('Interfaces fetch error:', e)
  }
  netLoading.value = false
}



const fetchAdslStatus = async () => {
  adslLoading.value = true
  try {
    const res = await GetADSLStatus(host.value, username.value, password.value, brand.value)
    adslStatus.value = res
  } catch (e: any) {
    console.error('ADSL fetch error:', e)
  }
  adslLoading.value = false
}

const rebootModem = async () => {
  if (!confirm('Are you sure you want to reboot the modem? Internet connection will drop.')) return
  loading.value = true
  action.value = 'reboot'
  setStatus('', false)
  try {
    await RebootModem(host.value, username.value, password.value, brand.value)
    setStatus('Reboot command sent! Please wait 1-2 minutes.', false)
  } catch (e: any) {
    setStatus(e.toString(), true)
  }
  loading.value = false
}

const fetchDevices = async () => {
  loading.value = true
  action.value = 'devices'
  try {
    const res = await GetConnectedDevices(host.value, username.value, password.value, brand.value)
    devices.value = res || []
  } catch (e: any) {
    setStatus('Failed to fetch devices: ' + e.toString(), true)
  }
  loading.value = false
}

const changeWifi = async () => {
  if (!newSsid.value) {
    setStatus('SSID cannot be empty', true)
    return
  }
  loading.value = true
  action.value = 'wifi'
  setStatus('', false)
  try {
    await ChangeWifiConfig(host.value, username.value, password.value, brand.value, newSsid.value, newWifiPass.value)
    setStatus('WiFi settings updated successfully. (Modem WiFi might restart)', false)
  } catch (e: any) {
    setStatus(e.toString(), true)
  }
  loading.value = false
}

const runCommand = async () => {
  if (!command.value) return
  loading.value = true
  action.value = 'cmd'
  output.value = ''
  try {
    const res = await RunModemCommand(host.value, username.value, password.value, command.value)
    output.value = res
  } catch (e: any) {
    output.value = 'Error: ' + e.toString()
  }
  loading.value = false
}
</script>

<style scoped>
.modem-container {
  padding: 24px;
  max-width: 900px;
  margin: 0 auto;
}

.header {
  margin-bottom: 24px;
}

.header h2 {
  font-size: 24px;
  font-weight: 600;
  color: var(--text);
  margin-bottom: 8px;
}

.subtitle {
  color: var(--text-muted);
  font-size: 14px;
}

.settings-card {
  background: var(--bg-card);
  border-radius: 12px;
  padding: 24px;
  border: 1px solid var(--border);
  box-shadow: var(--shadow-sm);
  margin-bottom: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
  margin-bottom: 4px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.row {
  display: flex;
  gap: 16px;
}
@media (max-width: 600px) {
  .row { flex-direction: column; }
}

.setting-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.flex-1 { flex: 1; }

.setting-group label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
}

.text-input {
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px 14px;
  color: var(--text);
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
}

.text-input:focus {
  border-color: var(--accent-down);
}

.command-input-row {
  display: flex;
  gap: 12px;
}

.btn-group {
  display: flex;
  gap: 12px;
  margin-top: 8px;
}

.primary-btn {
  background: var(--accent-down);
  color: #001018;
  border: none;
  border-radius: 8px;
  padding: 10px 24px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 120px;
}

.primary-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.primary-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pill-btn {
  background: var(--bg-card);
  border: 1px solid var(--border);
  color: var(--text);
  border-radius: 8px;
  padding: 8px 16px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.pill-btn:hover:not(:disabled) {
  background: var(--border);
}
.pill-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.danger-btn {
  color: var(--accent-error, #ff4444);
  border-color: rgba(255, 68, 68, 0.3);
}
.danger-btn:hover:not(:disabled) {
  background: rgba(255, 68, 68, 0.1);
  border-color: var(--accent-error, #ff4444);
}

.status-box {
  padding: 12px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
}
.status-box.success {
  background: rgba(0, 255, 136, 0.1);
  color: var(--accent-up);
}
.status-box.error {
  background: rgba(255, 68, 68, 0.1);
  color: var(--accent-error);
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}
.data-table th, .data-table td {
  padding: 12px;
  border-bottom: 1px solid var(--border);
  font-size: 14px;
  color: var(--text);
}
.data-table th {
  color: var(--text-muted);
  font-weight: 500;
}
.mono {
  font-family: monospace;
}
.empty-state {
  text-align: center;
  padding: 32px;
  color: var(--text-muted);
  font-size: 14px;
}

.terminal-output {
  background: #0d1117;
  color: #c9d1d9;
  padding: 16px;
  border-radius: 8px;
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
  overflow-x: auto;
  white-space: pre-wrap;
  word-wrap: break-word;
  margin-top: 16px;
}

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255,255,255,0.3);
  border-radius: 50%;
  border-top-color: #fff;
  animation: spin 1s linear infinite;
}
.spinner-dark {
  border-color: rgba(0,0,0,0.1);
  border-top-color: var(--text-color);
}
.spinner-red {
  border-color: rgba(255,71,87,0.3);
  border-top-color: #ff4757;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.adsl-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-top: 16px;
}

.adsl-box {
  background: var(--bg-elevated, #1a1e24);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 16px;
  transition: transform 0.2s, box-shadow 0.2s;
}

.adsl-box:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  border-color: var(--primary-color);
}

.adsl-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: bold;
}

.down-icon { background: rgba(0, 212, 255, 0.1); color: var(--accent-down); }
.up-icon { background: rgba(0, 255, 136, 0.1); color: var(--accent-up); }
.snr-icon { background: rgba(255, 165, 0, 0.1); color: var(--accent-warn); }
.att-icon { background: rgba(255, 68, 68, 0.1); color: var(--accent-error); }

.adsl-info {
  display: flex;
  flex-direction: column;
}

.adsl-label {
  font-size: 13px;
  color: var(--text-muted);
  font-weight: 500;
}

.adsl-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-color);
}
.adsl-value small {
  font-size: 12px;
  color: var(--text-muted);
  font-weight: 500;
}

.tabs-container {
  display: flex;
  gap: 8px;
  background: var(--bg-elevated, #2a3142);
  padding: 8px;
  border-radius: 12px;
  margin-bottom: 24px;
  overflow-x: auto;
  border: 1px solid var(--border-color, #3a4258);
}

.tabs-container::-webkit-scrollbar {
  display: none;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  background: transparent;
  color: var(--text-muted, #8b95a5);
  border: none;
  padding: 10px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.tab-btn .icon {
  font-size: 16px;
  opacity: 0.6;
  transition: opacity 0.2s ease;
}

.tab-btn:hover {
  color: var(--text-color, #e0e6ed);
  background: rgba(255, 255, 255, 0.05);
}

.tab-btn.active {
  background: var(--card-bg, #21262e);
  color: var(--text-color, #ffffff);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
  border: 1px solid var(--border-color, #3a4258);
}

.tab-btn.active .icon {
  opacity: 1;
}
</style>
