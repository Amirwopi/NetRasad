<template>
  <div class="iface-badge" :class="{ 'iface-badge--up': iface.isUp, 'iface-badge--down': !iface.isUp }">
    <span class="iface-badge__dot" />
    <span class="iface-badge__name">{{ iface.name }}</span>
    <span v-if="iface.isVPN" class="iface-badge__tag">VPN</span>
    <span v-else-if="iface.isVirtual" class="iface-badge__tag">VIRT</span>
    <span v-else-if="iface.isLoopback" class="iface-badge__tag">LO</span>
  </div>
</template>

<script setup lang="ts">
import type { NetworkInterface } from '@/types'
defineProps<{ iface: NetworkInterface }>()
</script>

<style scoped>
.iface-badge {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 4px 10px; border-radius: 20px;
  background: var(--bg-elevated); font-size: 12px;
}
.iface-badge__dot { width: 6px; height: 6px; border-radius: 50%; }
.iface-badge--up .iface-badge__dot { background: var(--accent-up); }
.iface-badge--down .iface-badge__dot { background: var(--text-dim); }
.iface-badge__name { max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.iface-badge__tag {
  font-size: 9px; font-weight: 700; padding: 1px 4px;
  border-radius: 3px; background: var(--border-strong); color: var(--text-muted);
}
</style>
