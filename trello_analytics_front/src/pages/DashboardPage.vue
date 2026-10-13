<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { fetchOverview } from '@/api/analytics'
import { fetchSyncStatus } from '@/api/sync'
import { messageFromUnknown } from '@/api/errors'
import { fetchThroughputSeries } from '@/composables/useSeriesWindows'
import {
  formatDurationSeconds,
  formatMetricCount,
  formatPercent,
} from '@/composables/format'
import type { OverviewResponse, SyncStatusResponse } from '@/types/api'
import type { SeriesPoint } from '@/composables/useSeriesWindows'
import { useFiltersStore } from '@/stores/filters'
import PeriodFilter from '@/components/PeriodFilter.vue'
import DataQualityBanner from '@/components/DataQualityBanner.vue'
import KpiCard from '@/components/KpiCard.vue'
import WorkloadBarChart from '@/components/charts/WorkloadBarChart.vue'
import ThroughputSeriesChart from '@/components/charts/ThroughputSeriesChart.vue'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'

const filters = useFiltersStore()
const { query } = storeToRefs(filters)

const loading = ref(true)
const error = ref<string | null>(null)
const overview = ref<OverviewResponse | null>(null)
const sync = ref<SyncStatusResponse | null>(null)
const seriesPoints = ref<SeriesPoint[] | null>(null)
const seriesBlocked = ref<'too_long' | 'empty' | null>(null)
const seriesError = ref<string | null>(null)
const seriesLoading = ref(false)

async function load() {
  loading.value = true
  error.value = null
  try {
    const [ov, st] = await Promise.all([
      fetchOverview(query.value),
      fetchSyncStatus().catch(() => null),
    ])
    overview.value = ov
    sync.value = st
  } catch (err) {
    overview.value = null
    error.value = messageFromUnknown(err)
  } finally {
    loading.value = false
  }
  await loadSeries()
}

async function loadSeries() {
  seriesLoading.value = true
  seriesPoints.value = null
  seriesBlocked.value = null
  seriesError.value = null
  try {
    const span = filters.inclusiveDaySpan()
    const res = await fetchThroughputSeries(query.value, span)
    if (!res.ok) {
      seriesBlocked.value = res.reason
      return
    }
    seriesPoints.value = res.points
  } catch (err) {
    // API failures must not look like Empty — separate error + retry.
    seriesError.value = messageFromUnknown(err)
  } finally {
    seriesLoading.value = false
  }
}

watch(query, () => void load(), { deep: true, immediate: true })

const kpis = computed(() => {
  const o = overview.value
  if (!o) return []
  return [
    {
      title: 'Пропускная способность',
      value: formatMetricCount(o.throughput.available, o.throughput.count),
      hint: o.throughput.available ? undefined : 'нет данных',
    },
    {
      title: 'Lead time (медиана)',
      value: formatDurationSeconds(o.lead_time.available, o.lead_time.median_seconds),
      hint: o.lead_time.available ? `n=${o.lead_time.count}` : 'нет данных',
    },
    {
      title: 'Cycle time (медиана)',
      value: formatDurationSeconds(o.cycle_time.available, o.cycle_time.median_seconds),
      hint: o.cycle_time.available ? `n=${o.cycle_time.count}` : 'нет данных',
    },
    {
      title: 'В работе (WIP)',
      value: formatMetricCount(o.wip.available, o.wip.count),
      hint: o.wip.available ? `заблокировано: ${o.wip.blocked_count}` : 'нет данных',
    },
    {
      title: 'Зависшие',
      value: formatMetricCount(o.stale.available, o.stale.count),
      hint: o.stale.available ? undefined : 'нет данных',
    },
    {
      title: 'Повторное открытие багов, %',
      value: formatPercent(o.bug_reopen.available, o.bug_reopen.reopen_percent),
      hint: o.bug_reopen.available
        ? `${o.bug_reopen.reopened_bugs}/${o.bug_reopen.completed_bugs}`
        : 'нет данных',
    },
  ]
})
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Дашборд</h1>
        <p class="text-sm text-muted-foreground">Метрики считаются на сервере за выбранный период.</p>
      </div>
      <PeriodFilter />
    </div>

    <DataQualityBanner :warnings="overview?.warnings" :sync="sync" />

    <Alert v-if="error" variant="destructive">
      <AlertTitle>Ошибка загрузки</AlertTitle>
      <AlertDescription class="flex items-center justify-between gap-3">
        <span>{{ error }}</span>
        <Button size="sm" variant="outline" @click="load">Повторить</Button>
      </AlertDescription>
    </Alert>

    <div v-if="loading" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <Skeleton v-for="i in 6" :key="i" class="h-28" />
    </div>

    <Empty v-else-if="!error && !overview">
      <EmptyHeader>
        <EmptyTitle>Нет данных</EmptyTitle>
        <EmptyDescription>Нет сводки за выбранный период.</EmptyDescription>
      </EmptyHeader>
    </Empty>

    <template v-else-if="overview">
      <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <KpiCard
          v-for="k in kpis"
          :key="k.title"
          :title="k.title"
          :value="k.value"
          :hint="k.hint"
        />
      </div>

      <div class="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle class="text-base">Нагрузка по участникам</CardTitle>
          </CardHeader>
          <CardContent>
            <WorkloadBarChart :workload="overview.workload" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle class="text-base">Пропускная способность по дням</CardTitle>
          </CardHeader>
          <CardContent>
            <Alert v-if="seriesError" variant="destructive">
              <AlertTitle>Ошибка ряда</AlertTitle>
              <AlertDescription class="flex items-center justify-between gap-3">
                <span>{{ seriesError }}</span>
                <Button size="sm" variant="outline" @click="loadSeries">Повторить</Button>
              </AlertDescription>
            </Alert>
            <Skeleton v-else-if="seriesLoading" class="h-72" />
            <ThroughputSeriesChart
              v-else
              :points="seriesPoints"
              :blocked-reason="seriesBlocked"
            />
          </CardContent>
        </Card>
      </div>
    </template>
  </div>
</template>
