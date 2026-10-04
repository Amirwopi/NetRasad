export interface InterfaceTraffic {
  interfaceId: string
  downloadBytes: number
  uploadBytes: number
  downloadRate: number
  uploadRate: number
  peakDownRate: number
  peakUpRate: number
  lastUpdate: string
}

export interface TrafficSnapshot {
  timestamp: string
  totalDownload: number
  totalUpload: number
  downloadRate: number
  uploadRate: number
  interfaces: Record<string, InterfaceTraffic>
}

export interface NetworkInterface {
  id: string
  name: string
  description: string
  hardwareAddr: string
  index: number
  type: number
  isUp: boolean
  isLoopback: boolean
  isVirtual: boolean
  isVPN: boolean
  mediaConnected: boolean
  ipAddresses: string[]
  gatewayIP: string
  ssid: string
}

export interface RateSample {
  timestamp: string
  downloadBps: number
  uploadBps: number
}

export interface TrafficRecord {
  id: number
  timestamp: string
  interfaceId: string
  downloadBytes: number
  uploadBytes: number
  category: number
}

export interface Capabilities {
  platform: string
  trafficMonitor: boolean
  processMonitor: boolean
  routerMonitor: boolean
  sync: boolean
  dbPath: string
}

export interface Metrics {
  samplesRead: number
  samplesDropped: number
  dbWrites: number
  dbErrors: number
  restarts: number
}

export interface ProcessTraffic {
  pid: number
  processName: string
  downloadBps: number
  uploadBps: number
  totalDownload: number
  totalUpload: number
  connCount: number
}

export interface QuotaConfig {
  enabled: boolean
  limitBytes: number
  period: 'daily' | 'weekly' | 'monthly'
  warningPercent: number
  resetDay: number
}


export type ThemeMode = 'dark' | 'light'
export type UnitSystem = 'binary' | 'decimal'
export type GraphColorScheme = 'citrus' | 'neon' | 'amber' | 'emerald' | 'dark' | 'classic' | 'custom'

export interface TaskbarOverlayConfig {
  downColorHex: string
  upColorHex: string
  bgColorHex: string
  transparentBg: boolean
  position: 'right' | 'left'
  width: number
  offsetX?: number
  monitor?: number
}

export interface AppSettings {
  theme: ThemeMode
  locale: 'en' | 'fa'
  units: UnitSystem
  pollIntervalMs: number
  graphMaxPoints: number
  showGrid: boolean
  graphColorScheme: GraphColorScheme
  graphBgColor: string
  graphDownColor: string
  graphUpColor: string
  graphTextColor: string
  graphGradient: boolean
  graphLineWidth: number
  displaySpeedInTaskbar: boolean
  taskbarTransparent: boolean
  taskbarPosition: 'right' | 'left'
  taskbarWidth: number
  taskbarOffsetX: number
  taskbarMonitor: number
  pingOverlayEnabled: boolean
  pingOverlayAddress: string
  pingOverlayLabel: string
  pingOverlayBgColorHex: string
  pingOverlayTextColorHex: string
  pingOverlayShape: 'circle' | 'rectangle'
  autoStart: boolean
  quota: QuotaConfig
}

export const DEFAULT_SETTINGS: AppSettings = {
  theme: 'dark',
  locale: 'fa',
  units: 'binary',
  pollIntervalMs: 1000,
  graphMaxPoints: 120,
  showGrid: true,
  graphColorScheme: 'citrus',
  graphBgColor: '#21262e',
  graphDownColor: '#ffa500',
  graphUpColor: '#8bc34a',
  graphTextColor: '#e0e6ed',
  graphGradient: true,
  graphLineWidth: 2,
  displaySpeedInTaskbar: true,
  taskbarTransparent: false,
  taskbarPosition: 'right',
  taskbarWidth: 165,
  taskbarOffsetX: 0,
  taskbarMonitor: 0,
  pingOverlayEnabled: false,
  pingOverlayAddress: '8.8.8.8',
  pingOverlayLabel: 'Google',
  pingOverlayBgColorHex: '#26201C',
  pingOverlayTextColorHex: '#AAAAAA',
  pingOverlayShape: 'rectangle',
  autoStart: false,
  quota: {
    enabled: true,
    limitBytes: 53687091200,
    period: 'monthly',
    warningPercent: 80,
    resetDay: 1,
  },
}


export interface PingResult {
  Host: string
  Sent: number
  Received: number
  PacketLoss: number
  MinRTT: number
  AvgRTT: number
  MaxRTT: number
  RTTs: number[]
  Output: string
}

export interface TracerouteHop {
  Hop: number
  Host: string
  IP: string
  RTT1: number
  RTT2: number
  RTT3: number
}

export interface TracerouteResult {
  Hops: TracerouteHop[]
  Output: string
}

export interface DNSResult {
  Host: string
  ARecords: string[]
  AAAARecords: string[]
  NSRecords: string[]
  MXRecords: string[]
  CNAME: string
}

export interface TCPResult {
  Host: string
  Port: number
  Success: boolean
  RTT: number
  Error: string
}

export interface GatewayInfo {
  GatewayIP: string
  Interface: string
  Metric: number
}


export type Protocol = 'tcp' | 'udp'

export type TCPState =
  | 'CLOSED'
  | 'LISTEN'
  | 'SYN_SENT'
  | 'SYN_RECEIVED'
  | 'ESTABLISHED'
  | 'FIN_WAIT1'
  | 'FIN_WAIT2'
  | 'CLOSE_WAIT'
  | 'CLOSING'
  | 'LAST_ACK'
  | 'TIME_WAIT'
  | 'DELETE_TCB'
  | 'UNKNOWN'
  | 'N/A'

export interface Connection {
  Protocol: Protocol
  LocalAddress: string
  LocalPort: number
  RemoteAddress: string
  RemotePort: number
  State: TCPState
  PID: number
  ProcessName: string
}
