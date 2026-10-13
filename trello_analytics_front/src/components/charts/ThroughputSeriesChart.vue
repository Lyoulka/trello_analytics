<script setup lang="ts">
import { computed } from 'vue'
import { Line } from 'vue-chartjs'
import {
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Title,
  Tooltip,
} from 'chart.js'
import type { SeriesPoint } from '@/composables/useSeriesWindows'
import { chartCssColor } from '@/composables/chartColors'
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend)

const props = defineProps<{
  points: SeriesPoint[] | null
  blockedReason?: 'too_long' | 'empty' | null
}>()

const chartData = computed(() => {
  const pts = props.points ?? []
  const color = chartCssColor(3)
  return {
    labels: pts.map((p) => p.label),
    datasets: [
      {
        label: 'Завершено за день',
        borderColor: color,
        backgroundColor: color,
        spanGaps: false,
        // Chart.js skips null — do not coerce to 0
        data: pts.map((p) => p.count),
      },
    ],
  }
})

const options = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: { legend: { position: 'bottom' as const } },
  scales: {
    y: { beginAtZero: true, ticks: { precision: 0 } },
  },
}
</script>

<template>
  <Empty v-if="blockedReason === 'too_long'">
    <EmptyHeader>
      <EmptyTitle>Ряд недоступен</EmptyTitle>
      <EmptyDescription>
        Период длиннее 31 дня. Выберите окно ≤31 день — time-series строится только через суточные
        вызовы API (без клиентской агрегации).
      </EmptyDescription>
    </EmptyHeader>
  </Empty>
  <Empty v-else-if="blockedReason === 'empty' || !points?.length">
    <EmptyHeader>
      <EmptyTitle>Нет точек ряда</EmptyTitle>
      <EmptyDescription>Не удалось построить дневные окна для выбранного периода.</EmptyDescription>
    </EmptyHeader>
  </Empty>
  <div v-else class="h-72">
    <Line :data="chartData" :options="options" />
  </div>
</template>
