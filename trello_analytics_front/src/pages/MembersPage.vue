<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchMembers, patchMember } from '@/api/members'
import { messageFromUnknown } from '@/api/errors'
import type { BoardMember } from '@/types/api'
import { useSessionStore } from '@/stores/session'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

const session = useSessionStore()
const canEdit = computed(() => session.hasMinRole('team_lead'))

const loading = ref(true)
const error = ref<string | null>(null)
const actionError = ref<string | null>(null)
const members = ref<BoardMember[]>([])
const busyId = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value = null
  try {
    const res = await fetchMembers()
    members.value = res.members ?? []
  } catch (err) {
    members.value = []
    error.value = messageFromUnknown(err)
  } finally {
    loading.value = false
  }
}

async function setActive(m: BoardMember, is_active: boolean) {
  if (!canEdit.value) return
  actionError.value = null
  busyId.value = m.id
  try {
    const updated = await patchMember(m.id, { is_active })
    members.value = members.value.map((row) => (row.id === updated.id ? updated : row))
  } catch (err) {
    actionError.value = messageFromUnknown(err)
  } finally {
    busyId.value = null
  }
}

function displayName(m: BoardMember) {
  return m.full_name || m.username || m.trello_member_id
}

onMounted(() => void load())
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Участники Trello</h1>
      <p class="text-sm text-muted-foreground">
        Неактивные участники не попадают в отчёты и графики (нагрузка, агрегаты по их карточкам).
      </p>
    </div>

    <Alert v-if="error" variant="destructive">
      <AlertTitle>Ошибка</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>
    <Alert v-if="actionError" variant="destructive">
      <AlertTitle>Не удалось обновить</AlertTitle>
      <AlertDescription>{{ actionError }}</AlertDescription>
    </Alert>

    <div v-if="loading" class="space-y-2">
      <Skeleton class="h-10 w-full" />
      <Skeleton class="h-10 w-full" />
      <Skeleton class="h-10 w-full" />
    </div>

    <Empty v-else-if="!error && members.length === 0">
      <EmptyHeader>
        <EmptyTitle>Нет участников</EmptyTitle>
        <EmptyDescription>Сначала выполните sync доски, чтобы импортировать участников.</EmptyDescription>
      </EmptyHeader>
    </Empty>

    <Card v-else-if="members.length">
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Имя</TableHead>
              <TableHead>Username</TableHead>
              <TableHead>В аналитике</TableHead>
              <TableHead v-if="canEdit" />
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="m in members" :key="m.id">
              <TableCell>{{ displayName(m) }}</TableCell>
              <TableCell class="text-muted-foreground">{{ m.username || '—' }}</TableCell>
              <TableCell>
                <Badge :variant="m.is_active ? 'success' : 'secondary'">
                  {{ m.is_active ? 'да' : 'нет' }}
                </Badge>
              </TableCell>
              <TableCell v-if="canEdit" class="text-right">
                <Button
                  v-if="m.is_active"
                  size="sm"
                  variant="outline"
                  :disabled="busyId === m.id"
                  @click="setActive(m, false)"
                >
                  Исключить
                </Button>
                <Button
                  v-else
                  size="sm"
                  variant="secondary"
                  :disabled="busyId === m.id"
                  @click="setActive(m, true)"
                >
                  Включить
                </Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  </div>
</template>
