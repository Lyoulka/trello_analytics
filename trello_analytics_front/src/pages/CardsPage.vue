<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ExternalLink } from 'lucide-vue-next'
import { fetchCards, trelloCardUrl } from '@/api/cards'
import { messageFromUnknown } from '@/api/errors'
import { formatDateTime } from '@/composables/format'
import type { CardListItem } from '@/types/api'
import CardHistoryDialog from '@/components/CardHistoryDialog.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const error = ref<string | null>(null)
const items = ref<CardListItem[]>([])
const total = ref(0)
const limit = ref(25)
const offset = ref(0)
const q = ref('')
const sort = ref<'name' | 'updated' | 'created'>('updated')
const order = ref<'asc' | 'desc'>('desc')
const archived = ref<'all' | 'false' | 'true'>('false')
const searchDraft = ref('')
const updatedFrom = ref('')
const updatedTo = ref('')
const historyOpen = ref(false)
const historyCardId = ref<string | null>(null)

function openHistory(cardId: string) {
  historyCardId.value = cardId
  historyOpen.value = true
  if (route.query.history !== cardId) {
    void router.replace({ query: { ...route.query, history: cardId } })
  }
}

function onHistoryOpenChange(open: boolean) {
  historyOpen.value = open
  if (!open) {
    historyCardId.value = null
    if (route.query.history != null) {
      const q = { ...route.query }
      delete q.history
      void router.replace({ query: q })
    }
  }
}

watch(
  () => route.query.history,
  (id) => {
    const historyId = typeof id === 'string' ? id : Array.isArray(id) ? id[0] : null
    if (historyId) {
      historyCardId.value = historyId
      historyOpen.value = true
    } else if (historyOpen.value) {
      historyOpen.value = false
      historyCardId.value = null
    }
  },
  { immediate: true },
)

function formatUpdated(c: CardListItem): string {
  const when = formatDateTime(c.last_trello_update_at)
  const who = c.last_updated_by_name?.trim()
  if (who) return `${when} · ${who}`
  return when
}

async function load() {
  loading.value = true
  error.value = null
  try {
    const res = await fetchCards({
      limit: limit.value,
      offset: offset.value,
      sort: sort.value,
      order: order.value,
      q: q.value || undefined,
      archived: archived.value === 'all' ? undefined : archived.value === 'true',
      updated_from: updatedFrom.value || undefined,
      updated_to: updatedTo.value || undefined,
    })
    items.value = res.items
    total.value = res.total
    limit.value = res.limit
    offset.value = res.offset
  } catch (err) {
    items.value = []
    error.value = messageFromUnknown(err)
  } finally {
    loading.value = false
  }
}

function applySearch() {
  q.value = searchDraft.value.trim()
  offset.value = 0
}

function applyUpdatedFilter() {
  offset.value = 0
  void load()
}

function clearUpdatedFilter() {
  updatedFrom.value = ''
  updatedTo.value = ''
  offset.value = 0
  void load()
}

function prevPage() {
  offset.value = Math.max(0, offset.value - limit.value)
}

function nextPage() {
  if (offset.value + limit.value < total.value) {
    offset.value += limit.value
  }
}

watch([offset, sort, order, archived, q], () => void load(), { immediate: true })
</script>

<template>
  <div class="space-y-4">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Карточки</h1>
      <p class="text-sm text-muted-foreground">Список карточек доски со ссылкой в Trello и историей.</p>
    </div>

    <Card>
      <CardHeader class="pb-3">
        <CardTitle class="text-base">Фильтры</CardTitle>
      </CardHeader>
      <CardContent class="flex flex-wrap items-end gap-3">
        <div class="space-y-1">
          <Label for="card-q">Поиск</Label>
          <div class="flex gap-2">
            <Input
              id="card-q"
              v-model="searchDraft"
              class="w-56"
              placeholder="имя карточки"
              @keyup.enter="applySearch"
            />
            <Button type="button" variant="secondary" @click="applySearch">Найти</Button>
          </div>
        </div>
        <div class="space-y-1">
          <Label for="upd-from">Обновлено с</Label>
          <Input id="upd-from" v-model="updatedFrom" type="date" class="w-40" />
        </div>
        <div class="space-y-1">
          <Label for="upd-to">Обновлено по (включ.)</Label>
          <Input id="upd-to" v-model="updatedTo" type="date" class="w-40" />
        </div>
        <div class="flex gap-2">
          <Button type="button" variant="secondary" @click="applyUpdatedFilter">Применить даты</Button>
          <Button
            v-if="updatedFrom || updatedTo"
            type="button"
            variant="ghost"
            @click="clearUpdatedFilter"
          >
            Сброс дат
          </Button>
        </div>
        <div class="space-y-1">
          <Label for="card-sort">Сортировка</Label>
          <select
            id="card-sort"
            v-model="sort"
            class="flex h-9 rounded-md border border-input bg-transparent px-3 text-sm"
          >
            <option value="updated">по обновлению</option>
            <option value="created">по созданию</option>
            <option value="name">по имени</option>
          </select>
        </div>
        <div class="space-y-1">
          <Label for="card-order">Порядок</Label>
          <select
            id="card-order"
            v-model="order"
            class="flex h-9 rounded-md border border-input bg-transparent px-3 text-sm"
          >
            <option value="desc">по убыванию</option>
            <option value="asc">по возрастанию</option>
          </select>
        </div>
        <div class="space-y-1">
          <Label for="card-arch">Архив</Label>
          <select
            id="card-arch"
            v-model="archived"
            class="flex h-9 rounded-md border border-input bg-transparent px-3 text-sm"
          >
            <option value="false">активные</option>
            <option value="true">архив</option>
            <option value="all">все</option>
          </select>
        </div>
      </CardContent>
    </Card>

    <Alert v-if="error" variant="destructive">
      <AlertTitle>Ошибка</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <div v-if="loading" class="space-y-2">
      <Skeleton class="h-10" />
      <Skeleton class="h-10" />
      <Skeleton class="h-10" />
    </div>

    <Empty v-else-if="!error && items.length === 0">
      <EmptyHeader>
        <EmptyTitle>Нет карточек</EmptyTitle>
        <EmptyDescription>По текущим фильтрам API вернул пустой список.</EmptyDescription>
      </EmptyHeader>
    </Empty>

    <Card v-else-if="items.length">
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Название</TableHead>
              <TableHead>Список</TableHead>
              <TableHead>Обновлено</TableHead>
              <TableHead>Архив</TableHead>
              <TableHead>Trello</TableHead>
              <TableHead>История</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="c in items" :key="c.id">
              <TableCell class="font-medium">{{ c.name }}</TableCell>
              <TableCell>{{ c.current_list_name ?? '—' }}</TableCell>
              <TableCell class="text-xs">
                {{ formatUpdated(c) }}
              </TableCell>
              <TableCell>
                <Badge :variant="c.is_archived ? 'secondary' : 'outline'">
                  {{ c.is_archived ? 'да' : 'нет' }}
                </Badge>
              </TableCell>
              <TableCell>
                <a
                  class="inline-flex items-center gap-1 text-sm text-primary underline-offset-4 hover:underline"
                  :href="trelloCardUrl(c.trello_card_id)"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  открыть
                  <ExternalLink class="size-3" />
                </a>
              </TableCell>
              <TableCell>
                <Button
                  type="button"
                  size="sm"
                  variant="link"
                  class="h-auto px-0"
                  @click="openHistory(c.id)"
                >
                  история
                </Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
      <div class="flex items-center justify-between border-t px-4 py-3 text-sm">
        <span class="text-muted-foreground">
          {{ offset + 1 }}–{{ Math.min(offset + items.length, total) }} из {{ total }}
        </span>
        <div class="flex gap-2">
          <Button size="sm" variant="outline" :disabled="offset <= 0" @click="prevPage">Назад</Button>
          <Button
            size="sm"
            variant="outline"
            :disabled="offset + limit >= total"
            @click="nextPage"
          >
            Далее
          </Button>
        </div>
      </div>
    </Card>

    <CardHistoryDialog
      :open="historyOpen"
      :card-id="historyCardId"
      @update:open="onHistoryOpenChange"
    />
  </div>
</template>
