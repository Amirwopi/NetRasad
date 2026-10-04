import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'dashboard',
    component: () => import('@/views/Dashboard.vue'),
    meta: { titleKey: 'nav.dashboard' },
  },
  {
    path: '/applications',
    name: 'applications',
    component: () => import('@/views/Applications.vue'),
    meta: { titleKey: 'nav.applications' },
  },
  {
    path: '/reports',
    name: 'reports',
    component: () => import('@/views/Reports.vue'),
    meta: { titleKey: 'nav.reports' },
  },
  {
    path: '/quota',
    name: 'quota',
    component: () => import('@/views/Quota.vue'),
    meta: { titleKey: 'nav.quota' },
  },
  {
    path: '/connections',
    name: 'connections',
    component: () => import('@/views/Connections.vue'),
    meta: { titleKey: 'nav.connections' },
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('@/views/Settings.vue'),
    meta: { titleKey: 'nav.settings' },
  },
  {
    path: '/diagnostics',
    name: 'diagnostics',
    component: () => import('@/views/Diagnostics.vue'),
    meta: { titleKey: 'nav.diagnostics' },
  },
  {
    path: '/modem',
    name: 'modem',
    component: () => import('@/views/Modem.vue'),
    meta: { titleKey: 'nav.modem' },
  },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 }
  },
})

router.afterEach((to) => {
  const key = (to.meta?.titleKey as string | undefined) ?? 'app.title'
  void key
})
