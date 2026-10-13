<script setup lang="ts">
/**
 * Screen → components map (task 2.0):
 * - Shell / nav: this layout (sidebar + top bar), Button, Separator-like borders
 * - Login: Field/Input/Label/Button (LoginPage)
 * - Dashboard: PeriodFilter, KpiCard/Card/Badge, DataQualityBanner, Skeleton/Empty/Alert, Chart.js
 * - Cards: Table + Input filters + Pagination buttons + Trello link
 * - Card history: modal Dialog on Cards (query ?history=)
 * - Reports: Card sections + copy Button
 * - Settings: Card forms, workflow Inputs, sync status Alert, DataQualityBanner
 * - Members (Member+ view / Team Lead+ toggle): Trello participants is_active
 * - Users (Admin): Table + create form Dialog-like Card
 * Primitives: Button, Card, Alert, Skeleton, Empty (+ Input/Label/Badge/Table)
 * No demo/mock routes in this shell.
 */
import { computed } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import {
  LayoutDashboard,
  ListTodo,
  FileText,
  Settings,
  Users,
  UserRound,
  Columns3,
  LogOut,
  Menu,
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { useSessionStore } from '@/stores/session'
import { ref } from 'vue'

const session = useSessionStore()
const { user } = storeToRefs(session)
const route = useRoute()
const router = useRouter()
const mobileOpen = ref(false)

const nav = computed(() => {
  const items = [
    { to: '/dashboard', label: 'Дашборд', icon: LayoutDashboard, minRole: 'member' as const },
    { to: '/cards', label: 'Карточки', icon: ListTodo, minRole: 'member' as const },
    { to: '/reports', label: 'Отчёты', icon: FileText, minRole: 'member' as const },
    { to: '/settings', label: 'Настройки', icon: Settings, minRole: 'member' as const },
    { to: '/members', label: 'Участники', icon: UserRound, minRole: 'member' as const },
    { to: '/lists', label: 'Колонки', icon: Columns3, minRole: 'member' as const },
    { to: '/users', label: 'Пользователи', icon: Users, minRole: 'admin' as const },
  ]
  return items.filter((i) => session.hasMinRole(i.minRole))
})

const roleLabel: Record<string, string> = {
  admin: 'Админ',
  team_lead: 'Тимлид',
  member: 'Участник',
}

async function onLogout() {
  await session.logout()
  await router.push({ name: 'login' })
}

function isActive(path: string) {
  return route.path === path || route.path.startsWith(path + '/')
}
</script>

<template>
  <div class="flex min-h-screen bg-background text-foreground">
    <!-- Sidebar -->
    <aside
      class="fixed inset-y-0 left-0 z-40 w-60 border-r bg-sidebar text-sidebar-foreground transition-transform md:static md:translate-x-0"
      :class="mobileOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'"
    >
      <div class="flex h-14 items-center border-b px-4 font-semibold tracking-tight">
        Trello Analytics
      </div>
      <nav class="flex flex-col gap-1 p-3">
        <RouterLink
          v-for="item in nav"
          :key="item.to"
          :to="item.to"
          class="flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors hover:bg-sidebar-accent"
          :class="isActive(item.to) ? 'bg-sidebar-accent font-medium' : 'text-muted-foreground'"
          @click="mobileOpen = false"
        >
          <component :is="item.icon" class="size-4" />
          {{ item.label }}
        </RouterLink>
      </nav>
    </aside>

    <div class="flex min-w-0 flex-1 flex-col">
      <header class="flex h-14 items-center justify-between gap-3 border-b px-4">
        <Button variant="ghost" size="icon" class="md:hidden" @click="mobileOpen = !mobileOpen">
          <Menu class="size-4" />
        </Button>
        <div class="min-w-0 flex-1 truncate text-sm text-muted-foreground">
          <span v-if="user">{{ user.display_name }} · {{ roleLabel[user.role] ?? user.role }}</span>
        </div>
        <Button variant="outline" size="sm" @click="onLogout">
          <LogOut class="size-4" />
          Выйти
        </Button>
      </header>
      <main class="flex-1 p-4 md:p-6">
        <RouterView />
      </main>
    </div>
  </div>
</template>
