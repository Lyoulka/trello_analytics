import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import type { AppRole } from '@/types/roles'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    minRole?: AppRole
    public?: boolean
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('../pages/LoginPage.vue'),
    meta: { public: true },
  },
  {
    path: '/',
    component: () => import('../layouts/AppShell.vue'),
    meta: { requiresAuth: true, minRole: 'member' },
    children: [
      { path: '', redirect: '/dashboard' },
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('../pages/DashboardPage.vue'),
        meta: { requiresAuth: true, minRole: 'member' },
      },
      {
        path: 'cards',
        name: 'cards',
        component: () => import('../pages/CardsPage.vue'),
        meta: { requiresAuth: true, minRole: 'member' },
      },
      {
        path: 'cards/:id/history',
        name: 'card-history',
        component: () => import('../pages/CardHistoryPage.vue'),
        meta: { requiresAuth: true, minRole: 'member' },
      },
      {
        path: 'reports',
        name: 'reports',
        component: () => import('../pages/ReportsPage.vue'),
        meta: { requiresAuth: true, minRole: 'member' },
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('../pages/SettingsPage.vue'),
        meta: { requiresAuth: true, minRole: 'member' },
      },
      {
        path: 'members',
        name: 'members',
        component: () => import('../pages/MembersPage.vue'),
        meta: { requiresAuth: true, minRole: 'member' },
      },
      {
        path: 'lists',
        name: 'lists',
        component: () => import('../pages/ListsPage.vue'),
        meta: { requiresAuth: true, minRole: 'member' },
      },
      {
        path: 'users',
        name: 'users',
        component: () => import('../pages/UsersPage.vue'),
        meta: { requiresAuth: true, minRole: 'admin' },
      },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const session = useSessionStore()

  if (!session.ready) {
    await session.refreshMe()
  }

  if (to.meta.public) {
    if (session.isAuthenticated && to.name === 'login') {
      return { path: '/dashboard' }
    }
    return true
  }

  const requiresAuth = to.matched.some((r) => r.meta.requiresAuth)
  if (requiresAuth && !session.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  const minRole = [...to.matched].reverse().find((r) => r.meta.minRole)?.meta.minRole
  if (minRole && !session.hasMinRole(minRole)) {
    return { path: '/dashboard' }
  }

  return true
})
