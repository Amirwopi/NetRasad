import { ref } from 'vue'
import * as AppBinding from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import type {
  TrafficSnapshot,
  NetworkInterface,
  RateSample,
  TrafficRecord,
  Capabilities,
  Metrics,
  PingResult,
  TracerouteResult,
  DNSResult,
  TCPResult,
  GatewayInfo,
  Connection,
  ProcessTraffic,
} from '@/types'

export function useWails() {
  const isWails = ref<boolean>(
    typeof window !== 'undefined' && !!window.runtime,
  )

  function on(event: string, cb: (data: any) => void): () => void {
    if (!isWails.value) return () => {}
    EventsOn(event, cb)
    return () => {
      try { EventsOff(event) } catch {  }
    }
  }

  function assertWails(): void {
    if (!isWails.value) {
      throw new Error('Wails runtime not available (running in browser)')
    }
  }


  async function getDashboardState(): Promise<TrafficSnapshot> {
    assertWails()
    return AppBinding.GetDashboardState() as unknown as Promise<TrafficSnapshot>
  }

  async function getRateHistory(): Promise<RateSample[]> {
    assertWails()
    return AppBinding.GetRateHistory() as unknown as Promise<RateSample[]>
  }

  async function getInterfaces(): Promise<NetworkInterface[]> {
    assertWails()
    return AppBinding.GetInterfaces() as unknown as Promise<NetworkInterface[]>
  }

  async function getActiveInterfaces(): Promise<NetworkInterface[]> {
    assertWails()
    return AppBinding.GetActiveInterfaces() as unknown as Promise<NetworkInterface[]>
  }

  async function setSelectedInterfaces(ids: string[]): Promise<void> {
    assertWails()
    await AppBinding.SetSelectedInterfaces(ids)
  }

  async function getSelectedInterfaces(): Promise<string[]> {
    assertWails()
    return AppBinding.GetSelectedInterfaces() as unknown as Promise<string[]>
  }

  async function getMetrics(): Promise<Metrics> {
    assertWails()
    const raw = await AppBinding.GetMetrics() as Record<string, number>
    return {
      samplesRead: raw['samplesRead'] ?? 0,
      samplesDropped: raw['samplesDropped'] ?? 0,
      dbWrites: raw['dbWrites'] ?? 0,
      dbErrors: raw['dbErrors'] ?? 0,
      restarts: raw['restarts'] ?? 0,
    }
  }

  async function getCapabilities(): Promise<Capabilities> {
    assertWails()
    const raw = await AppBinding.GetCapabilities() as Record<string, any>
    return {
      platform: raw['platform'] ?? 'unknown',
      trafficMonitor: raw['trafficMonitor'] ?? false,
      processMonitor: raw['processMonitor'] ?? false,
      routerMonitor: raw['routerMonitor'] ?? false,
      sync: raw['sync'] ?? false,
      dbPath: raw['dbPath'] ?? '',
    }
  }

  async function checkDatabase(): Promise<string | null> {
    assertWails()
    await AppBinding.CheckDatabase()
    return 'OK'
  }

  async function getTrafficHistory(
    start: string,
    end: string,
    res: string,
  ): Promise<TrafficRecord[]> {
    assertWails()
    const raw = await AppBinding.GetTrafficHistory(start, end, res) as any[]
    return raw.map((s) => ({
      id: s.id ?? 0,
      timestamp: typeof s.timestamp === 'string' ? s.timestamp : String(s.timestamp),
      interfaceId: s.interfaceId ?? '',
      downloadBytes: s.downloadBytes ?? 0,
      uploadBytes: s.uploadBytes ?? 0,
      category: Number(s.category ?? 0),
    }))
  }

  function onTrafficUpdated(cb: (data: TrafficSnapshot) => void): () => void {
    return on('traffic:updated', cb)
  }


  async function ping(host: string, count: number = 4): Promise<PingResult> {
    assertWails()
    return AppBinding.Ping(host, count) as unknown as Promise<PingResult>
  }

  async function traceroute(host: string, maxHops: number = 30): Promise<TracerouteResult> {
    assertWails()
    return AppBinding.Traceroute(host, maxHops) as unknown as Promise<TracerouteResult>
  }

  async function dnsLookup(host: string): Promise<DNSResult> {
    assertWails()
    return AppBinding.DNSLookup(host) as unknown as Promise<DNSResult>
  }

  async function tcpConnect(host: string, port: number, timeoutSec: number = 5): Promise<TCPResult> {
    assertWails()
    return AppBinding.TCPConnect(host, port, timeoutSec) as unknown as Promise<TCPResult>
  }

  async function getDefaultGateway(): Promise<GatewayInfo> {
    assertWails()
    return AppBinding.GetDefaultGateway() as unknown as Promise<GatewayInfo>
  }


  async function getConnections(): Promise<Connection[]> {
    assertWails()
    return AppBinding.GetConnections() as unknown as Promise<Connection[]>
  }

  async function getProcessTraffic(): Promise<ProcessTraffic[]> {
    assertWails()
    return AppBinding.GetProcessTraffic() as unknown as Promise<ProcessTraffic[]>
  }

  function onProcessTrafficUpdated(cb: (data: ProcessTraffic[]) => void): () => void {
    return on('traffic:process_updated', cb)
  }

  async function setTaskbarWidgetEnabled(enabled: boolean): Promise<void> {
    if (!isWails.value) return
    try {
      if ((AppBinding as any).SetTaskbarWidgetEnabled) {
        await (AppBinding as any).SetTaskbarWidgetEnabled(enabled)
      }
    } catch {}
  }

  async function getTaskbarWidgetEnabled(): Promise<boolean> {
    if (!isWails.value) return false
    try {
      if ((AppBinding as any).GetTaskbarWidgetEnabled) {
        return await (AppBinding as any).GetTaskbarWidgetEnabled()
      }
    } catch {}
    return false
  }

  async function setTaskbarConfig(cfg: any): Promise<void> {
    if (!isWails.value) return
    try {
      if ((AppBinding as any).SetTaskbarConfig) {
        await (AppBinding as any).SetTaskbarConfig(cfg)
      }
    } catch {}
  }

  async function getPingOverlayConfig(): Promise<any> {
    if (!isWails.value) return null
    try {
      if ((AppBinding as any).GetPingOverlayConfig) {
        return await (AppBinding as any).GetPingOverlayConfig()
      }
    } catch {}
    return null
  }

  async function setPingOverlayConfig(cfg: any): Promise<void> {
    if (!isWails.value) return
    try {
      if ((AppBinding as any).SetPingOverlayConfig) {
        await (AppBinding as any).SetPingOverlayConfig(cfg)
      }
    } catch {}
  }

  async function setAutoStart(enable: boolean): Promise<void> {
    if (!isWails.value) return
    try {
      if ((AppBinding as any).SetAutoStart) {
        await (AppBinding as any).SetAutoStart(enable)
      }
    } catch {}
  }

  async function isAutoStartEnabled(): Promise<boolean> {
    if (!isWails.value) return false
    try {
      if ((AppBinding as any).IsAutoStartEnabled) {
        return await (AppBinding as any).IsAutoStartEnabled()
      }
    } catch {}
    return false
  }

  return {
    isWails,
    on,
    getDashboardState,
    getRateHistory,
    getInterfaces,
    getActiveInterfaces,
    setSelectedInterfaces,
    getSelectedInterfaces,
    getMetrics,
    getCapabilities,
    checkDatabase,
    getTrafficHistory,
    onTrafficUpdated,
    ping,
    traceroute,
    dnsLookup,
    tcpConnect,
    getDefaultGateway,
    getConnections,
    getProcessTraffic,
    onProcessTrafficUpdated,
    setTaskbarWidgetEnabled,
    getTaskbarWidgetEnabled,
    setTaskbarConfig,
    getPingOverlayConfig,
    setPingOverlayConfig,
    setAutoStart,
    isAutoStartEnabled,
  }
}
