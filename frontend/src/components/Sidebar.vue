<template>
  <nav class="sidebar">
    <div class="sidebar__logo">
      <img src="@/assets/logo.svg" alt="NetRasad" class="sidebar__logo-img" />
    </div>
    <ul class="sidebar__nav">
      <li v-for="item in navItems" :key="item.to">
        <router-link :to="item.to" class="sidebar__link">
          <span class="sidebar__icon" v-html="item.icon" />
          <span>{{ $t(item.label) }}</span>
        </router-link>
      </li>
    </ul>
    <div class="sidebar__footer">
      <button class="sidebar__locale" @click="settings.toggleLocale">
        {{ settings.locale === 'en' ? 'FA' : 'EN' }}
      </button>
      <button class="sidebar__locale" @click="settings.toggleTheme">
        {{ settings.theme === 'dark' ? '☀' : '☾' }}
      </button>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { useSettingsStore } from '@/stores/settings'

const settings = useSettingsStore()

const icons = {
  dashboard: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="8" height="8" rx="1"/><rect x="13" y="3" width="8" height="8" rx="1"/><rect x="3" y="13" width="8" height="8" rx="1"/><rect x="13" y="13" width="8" height="8" rx="1"/></svg>',
  applications: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="4" width="16" height="16" rx="2"/><circle cx="9" cy="9" r="2"/><path d="M15 15l-3-3"/></svg>',
  reports: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 3v18h18"/><path d="M7 14l4-4 4 4 5-5"/></svg>',
  quota: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>',
  connections: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/></svg>',
  diagnostics: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>',
  settings: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>',
}

const navItems = [
  { to: '/', label: 'nav.dashboard', icon: icons.dashboard },
  { to: '/applications', label: 'nav.applications', icon: icons.applications },
  { to: '/reports', label: 'nav.reports', icon: icons.reports },
  { to: '/quota', label: 'nav.quota', icon: icons.quota },
  { to: '/connections', label: 'nav.connections', icon: icons.connections },
  { to: '/diagnostics', label: 'nav.diagnostics', icon: icons.diagnostics },
  { to: '/settings', label: 'nav.settings', icon: icons.settings },
]
</script>

<style scoped>
.sidebar {
  width: var(--sidebar-width);
  background: var(--bg-card);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}
html[dir='rtl'] .sidebar { border-right: none; border-left: 1px solid var(--border); }
.sidebar__logo {
  display: flex; align-items: center; justify-content: center;
  padding: 16px 14px; border-bottom: 1px solid var(--border);
}
.sidebar__logo-img { height: 36px; width: auto; }
.sidebar__nav { list-style: none; flex: 1; padding: 8px 0; }
.sidebar__link {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 16px; color: var(--text-muted);
  font-size: 13px; font-weight: 500;
  transition: color var(--transition), background var(--transition);
}
.sidebar__link:hover { color: var(--text); background: var(--bg-card-hover); }
.sidebar__link.router-link-active { color: var(--accent-down); background: rgba(0, 212, 255, 0.08); }
.sidebar__icon { display: flex; align-items: center; }
.sidebar__footer {
  display: flex; gap: 8px; padding: 12px 14px;
  border-top: 1px solid var(--border);
}
.sidebar__locale {
  width: 32px; height: 32px; border-radius: var(--radius-sm);
  border: 1px solid var(--border); background: var(--bg-elevated);
  color: var(--text-muted); font-size: 12px; font-weight: 600;
  display: flex; align-items: center; justify-content: center;
  transition: border-color var(--transition);
}
.sidebar__locale:hover { border-color: var(--accent-down); color: var(--text); }
</style>
