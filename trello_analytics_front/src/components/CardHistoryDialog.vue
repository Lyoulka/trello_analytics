<script setup lang="ts">
import { ref, watch } from 'vue'
import { ExternalLink, X } from 'lucide-vue-next'
import {
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
} from 'reka-ui'
import { fetchCard, fetchCardHistory, trelloCardUrl } from '@/api/cards'
import { messageFromUnknown } from '@/api/errors'
import { formatDateTime } from '@/composables/format'
import type { CardDetail, CardHistoryItem } from '@/types/api'
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

const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
  cardId: string | null
}>()

const loading = ref(false)
const error = ref<string | null>(null)
const card = ref<CardDetail | null>(null)
const events = ref<CardHistoryItem[]>([])

async function load(id: string) {
  loading.value = true
  error.value = null
  try {
    const [detail, history] = await Promise.all([fetchCard(id), fetchCardHistory(id)])
    card.value = detail
    events.value = history.events ?? []
  } catch (err) {
    card.value = null
    events.value = []
    error.value = messageFromUnknown(err)
  } finally {
    loading.value = false
  }
}

watch(
  () => [open.value, props.cardId] as const,
  ([isOpen, id]) => {
    if (isOpen && id) {
      void load(id)
      return
    }
    if (!isOpen) {
      card.value = null
      events.value = []
      error.value = null
    }
  },
  { immediate: true },
)

function formatListMove(e: CardHistoryItem): string {
  const from = e.source_list_name?.trim() || null
  const to = e.destination_list_name?.trim() || null
  if (!from && !to) return '—'
  return `${from ?? '—'} → ${to ?? '—'}`
}

function formatMember(e: CardHistoryItem): string {
  return e.member_name?.trim() || '—'
}
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-in data-[state=closed]:animate-out" />
      <DialogContent
        class="fixed top-1/2 left-1/2 z-50 flex max-h-[85vh] w-[min(56rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 flex-col gap-4 rounded-lg border bg-background p-6 shadow-lg outline-none"
        @open-auto-focus.prevent
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0 space-y-1">
            <DialogTitle class="text-lg font-semibold tracking-tight">
              {{ card?.name ?? 'История карточки' }}
            </DialogTitle>
            <DialogDescription class="text-sm text-muted-foreground">
              <template v-if="card">
                Список: {{ card.current_list_name ?? '—' }} · ID Trello: {{ card.trello_card_id }}
              </template>
              <template v-else>Хронология событий карточки</template>
            </DialogDescription>
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <a
              v-if="card"
              class="inline-flex items-center gap-1 text-sm text-primary underline-offset-4 hover:underline"
              :href="trelloCardUrl(card.trello_card_id)"
              target="_blank"
              rel="noopener noreferrer"
            >
              Trello
              <ExternalLink class="size-3" />
            </a>
            <DialogClose
              class="inline-flex size-9 items-center justify-center rounded-md hover:bg-accent"
              aria-label="Закрыть"
            >
              <X class="size-4" />
            </DialogClose>
          </div>
        </div>

        <Alert v-if="error" variant="destructive">
          <AlertTitle>Ошибка</AlertTitle>
          <AlertDescription>{{ error }}</AlertDescription>
        </Alert>

        <div v-if="loading" class="space-y-2">
          <Skeleton class="h-10" />
          <Skeleton class="h-10" />
          <Skeleton class="h-10" />
        </div>

        <Empty v-else-if="!error && events.length === 0">
          <EmptyHeader>
            <EmptyTitle>Нет событий</EmptyTitle>
            <EmptyDescription>
              История пуста или неполная (импортируется лента доски).
            </EmptyDescription>
          </EmptyHeader>
        </Empty>

        <div v-else-if="events.length" class="min-h-0 flex-1 overflow-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Дата</TableHead>
                <TableHead>Тип</TableHead>
                <TableHead>Откуда → куда</TableHead>
                <TableHead>Участник</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="e in events" :key="e.id">
                <TableCell class="whitespace-nowrap text-xs">{{ formatDateTime(e.action_date) }}</TableCell>
                <TableCell>{{ e.action_type }}</TableCell>
                <TableCell class="text-xs text-muted-foreground">
                  {{ formatListMove(e) }}
                </TableCell>
                <TableCell class="text-xs">{{ formatMember(e) }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
