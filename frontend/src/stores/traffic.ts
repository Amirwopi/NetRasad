import { defineStore } from 'pinia'
import { ref, computed, shallowRef } from 'vue'
import type {
  TrafficSnapshot,
  NetworkInterface,
  RateSample,
  InterfaceTraffic,
} from '@/types'

export const useTrafficStore = defineStore('traffic', () => {
  const snapshot = shallowRef<TrafficSnapshot | null>(null)
  const rateHistory = ref<RateSample[]>([])
  const interfaces = ref<NetworkInterface[]>([])
  const lastError = ref<string | null>(null)
  const lastUpdated = ref<number>(0)
  const isLive = ref(false)

  const downloadRate = computed(() => snapshot.value?.downloadRate ?? 0)
  const uploadRate = computed(() => snapshot.value?.uploadRate ?? 0)
  const historicalDownload = ref<number>(0)
  const historicalUpload = ref<number>(0)

  const totalDownload = computed(() => (snapshot.value?.totalDownload ?? 0) + historicalDownload.value)
  const totalUpload = computed(() => (snapshot.value?.totalUpload ?? 0) + historicalUpload.value)

  const interfaceList = computed<InterfaceTraffic[]>(() => {
    const map = snapshot.value?.interfaces
    if (!map) return []
    return Object.values(map).sort(
      (a, b) => b.downloadRate + b.uploadRate - (a.downloadRate + a.uploadRate),
    )
  })

  const topInterfaces = computed(() => interfaceList.value.slice(0, 6))

  function setSnapshot(s: TrafficSnapshot): void {
    snapshot.value = s
    lastUpdated.value = Date.now()
    isLive.value = true
    rateHistory.value.push({
      timestamp: s.timestamp,
      downloadBps: s.downloadRate,
      uploadBps: s.uploadRate,
    })
    if (rateHistory.value.length > 600) {
      rateHistory.value.splice(0, rateHistory.value.length - 600)
    }
  }

  function setRateHistory(samples: RateSample[]): void {
    rateHistory.value = samples.slice(-600)
  }

  function setInterfaces(list: NetworkInterface[]): void {
    interfaces.value = list
  }

  function setError(msg: string | null): void {
    lastError.value = msg
  }

  function setLive(v: boolean): void {
    isLive.value = v
  }

  function reset(): void {
    snapshot.value = null
    rateHistory.value = []
    interfaces.value = []
    lastError.value = null
    lastUpdated.value = 0
    isLive.value = false
  }

  return {
    snapshot,
    rateHistory,
    interfaces,
    lastError,
    lastUpdated,
    isLive,
    downloadRate,
    uploadRate,
    totalDownload,
    totalUpload,
    historicalDownload,
    historicalUpload,
    interfaceList,
    topInterfaces,
    setSnapshot,
    setRateHistory,
    setInterfaces,
    setError,
    setLive,
    reset,
  }
})
