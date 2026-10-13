<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchLists, patchList } from '@/api/lists'
import { messageFromUnknown } from '@/api/errors'
import type { BoardList } from '@/types/api'
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
const lists = ref<BoardList[]>([])
const busyId = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value = null
  try {
    const res = await fetchLists()
    lists.value = res.lists ?? []
  } catch (err) {
    lists.value = []
    error.value = messageFromUnknown(err)
  } finally {
    loading.value = false
  }
}

async function setActive(l: BoardList, is_active: boolean) {
  if (!canEdit.value) return
  actionError.value = null
  busyId.value = l.id
  try {
    const updated = await patchList(l.id, { is_active })
    lists.value = lists.value.map((row) => (row.id === updated.id ? updated : row))
  } catch (err) {
    actionError.value = messageFromUnknown(err)
  } finally {
    busyId.value = null
  }
}

onMounted(() => void load())
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Колонки Trello</h1>
      <p class="text-sm text-muted-foreground">
        Неактивные колонки исключают карточки, которые сейчас в них лежат, из отчётов и графиков.
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
    </div>

    <Empty v-else-if="!error && lists.length === 0">
      <EmptyHeader>
        <EmptyTitle>Нет колонок</EmptyTitle>
        <EmptyDescription>Сначала выполните sync доски.</EmptyDescription>
      </EmptyHeader>
    </Empty>

    <Card v-else-if="lists.length">
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Название</TableHead>
              <TableHead>Стадия</TableHead>
              <TableHead>Архив</TableHead>
              <TableHead>В аналитике</TableHead>
              <TableHead v-if="canEdit" />
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="l in lists" :key="l.id">
              <TableCell class="font-medium">{{ l.name }}</TableCell>
              <TableCell class="text-muted-foreground">{{ l.logical_stage || '—' }}</TableCell>
              <TableCell>
                <Badge :variant="l.is_archived ? 'secondary' : 'outline'">
                  {{ l.is_archived ? 'да' : 'нет' }}
                </Badge>
              </TableCell>
              <TableCell>
                <Badge :variant="l.is_active ? 'success' : 'secondary'">
                  {{ l.is_active ? 'да' : 'нет' }}
                </Badge>
              </TableCell>
              <TableCell v-if="canEdit" class="text-right">
                <Button
                  v-if="l.is_active"
                  size="sm"
                  variant="outline"
                  :disabled="busyId === l.id"
                  @click="setActive(l, false)"
                >
                  Исключить
                </Button>
                <Button
                  v-else
                  size="sm"
                  variant="secondary"
                  :disabled="busyId === l.id"
                  @click="setActive(l, true)"
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
