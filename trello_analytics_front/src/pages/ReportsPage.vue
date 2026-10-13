<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { ExternalLink } from 'lucide-vue-next'
import { fetchReportPreview, formatReportMetric, reportToPlainText } from '@/api/reports'
import { trelloCardUrl } from '@/api/cards'
import { messageFromUnknown } from '@/api/errors'
import type { ReportCard, ReportPreview } from '@/types/api'
import { useFiltersStore } from '@/stores/filters'
import PeriodFilter from '@/components/PeriodFilter.vue'
import DataQualityBanner from '@/components/DataQualityBanner.vue'
import KpiCard from '@/components/KpiCard.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { Badge } from '@/components/ui/badge'

const filters = useFiltersStore()
const { query } = storeToRefs(filters)

const loading = ref(true)
const error = ref<string | null>(null)
const report = ref<ReportPreview | null>(null)
const copyState = ref<'idle' | 'ok' | 'fail'>('idle')

function metricCard(title: string, value: number | null | undefined) {
  const available = value !== null && value !== undefined
  return {
    title,
    value: formatReportMetric(value),
    hint: available ? undefined : 'нет данных',
  }
}

const summaryKpis = computed(() => {
  const s = report.value?.summary
  if (!s) return []
  return [
    metricCard('Завершено', s.completed_count),
    metricCard('Багов исправлено', s.bugs_fixed_count),
    metricCard('Возвратов', s.reopened_count),
    metricCard('В работе (WIP)', s.wip_count),
  ]
})

const deltaKpis = computed(() => {
  const d = report.value?.deltas
  if (!d) return []
  return [
    metricCard('Δ завершённых', d.completed_delta),
    metricCard('Δ возвратов', d.reopened_delta),
  ]
})

async function load() {
  loading.value = true
  error.value = null
  copyState.value = 'idle'
  try {
    report.value = await fetchReportPreview(query.value)
  } catch (err) {
    report.value = null
    error.value = messageFromUnknown(err)
  } finally {
    loading.value = false
  }
}

async function copyText() {
  if (!report.value) return
  const text = reportToPlainText(report.value)
  try {
    await navigator.clipboard.writeText(text)
    copyState.value = 'ok'
  } catch {
    copyState.value = 'fail'
  }
}

watch(query, () => void load(), { deep: true, immediate: true })

function memberLabel(c: ReportCard): string {
  const names = (c.member_names ?? []).map((n) => n.trim()).filter(Boolean)
  return names.length ? names.join(', ') : '—'
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Отчёты</h1>
        <p class="text-sm text-muted-foreground">Превью отчёта за период; можно скопировать текстом.</p>
      </div>
      <PeriodFilter show-member />
    </div>

    <DataQualityBanner :warnings="report?.warnings" />

    <div class="flex flex-wrap gap-2">
      <Button :disabled="!report || loading" @click="copyText">Копировать текст</Button>
      <Badge v-if="copyState === 'ok'" variant="success">Скопировано</Badge>
      <Badge v-else-if="copyState === 'fail'" variant="destructive">Не удалось скопировать</Badge>
    </div>

    <Alert v-if="error" variant="destructive">
      <AlertTitle>Ошибка</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <div v-if="loading" class="grid gap-4 md:grid-cols-2">
      <Skeleton class="h-40" />
      <Skeleton class="h-40" />
    </div>

    <Empty v-else-if="!error && !report">
      <EmptyHeader>
        <EmptyTitle>Нет отчёта</EmptyTitle>
        <EmptyDescription>Нет данных отчёта за выбранный период.</EmptyDescription>
      </EmptyHeader>
    </Empty>

    <template v-else-if="report">
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <KpiCard
          v-for="kpi in summaryKpis"
          :key="kpi.title"
          :title="kpi.title"
          :value="kpi.value"
          :hint="kpi.hint"
        />
      </div>

      <div v-if="deltaKpis.length" class="grid gap-4 md:grid-cols-2">
        <KpiCard
          v-for="kpi in deltaKpis"
          :key="kpi.title"
          :title="kpi.title"
          :value="kpi.value"
          :hint="kpi.hint"
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle class="text-base">Завершённые задачи</CardTitle>
          <CardDescription>{{ report.completed_tasks.length }} шт.</CardDescription>
        </CardHeader>
        <CardContent>
          <ul v-if="report.completed_tasks.length" class="divide-y text-sm">
            <li
              v-for="c in report.completed_tasks"
              :key="c.id || c.trello_card_id"
              class="flex flex-wrap items-baseline justify-between gap-2 py-2 first:pt-0 last:pb-0"
            >
              <div class="min-w-0">
                <div class="font-medium">{{ c.name }}</div>
                <div class="text-xs text-muted-foreground">Участник: {{ memberLabel(c) }}</div>
              </div>
              <a
                class="inline-flex shrink-0 items-center gap-1 text-sm text-primary underline-offset-4 hover:underline"
                :href="trelloCardUrl(c.trello_card_id)"
                target="_blank"
                rel="noopener noreferrer"
              >
                Trello
                <ExternalLink class="size-3" />
              </a>
            </li>
          </ul>
          <p v-else class="text-sm text-muted-foreground">Список пуст.</p>
        </CardContent>
      </Card>

      <div class="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle class="text-base">Баги</CardTitle>
          </CardHeader>
          <CardContent class="space-y-2 text-sm">
            <p>Исправлено: {{ report.bugs.fixed.length }}</p>
            <p>Снова открыто: {{ report.bugs.reopened.length }}</p>
            <p>Ещё в работе: {{ report.bugs.still_in_work.length }}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle class="text-base">Текущая работа</CardTitle>
          </CardHeader>
          <CardContent class="space-y-2 text-sm">
            <p>В работе (WIP): {{ report.current_work.wip.length }}</p>
            <p>Зависшие: {{ report.current_work.stale.length }}</p>
          </CardContent>
        </Card>
      </div>
    </template>
  </div>
</template>
