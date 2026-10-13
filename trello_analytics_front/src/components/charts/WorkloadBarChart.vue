<script setup lang="ts">
import { computed } from 'vue'
import { Bar } from 'vue-chartjs'
import {
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LinearScale,
  Title,
  Tooltip,
} from 'chart.js'
import type { WorkloadEntry } from '@/types/api'
import { chartCssColor } from '@/composables/chartColors'
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'

ChartJS.register(CategoryScale, LinearScale, BarElement, Title, Tooltip, Legend)

const props = defineProps<{
  workload: WorkloadEntry[] | null
}>()

const hasData = computed(() => (props.workload?.length ?? 0) > 0)

const chartData = computed(() => {
  const rows = props.workload ?? []
  const color1 = chartCssColor(1)
  const color2 = chartCssColor(2)
  return {
    labels: rows.map((r) => r.full_name?.trim() || r.member_id.slice(0, 8)),
    datasets: [
      {
        label: 'Активные',
        backgroundColor: color1,
        data: rows.map((r) => r.active_cards),
      },
      {
        label: 'Завершённые',
        backgroundColor: color2,
        data: rows.map((r) => r.completed_cards),
      },
    ],
  }
})

const options = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { position: 'bottom' as const },
  },
  scales: {
    y: { beginAtZero: true, ticks: { precision: 0 } },
  },
}
</script>

<template>
  <Empty v-if="!hasData">
    <EmptyHeader>
      <EmptyTitle>Нет данных по нагрузке</EmptyTitle>
      <EmptyDescription>Нет записей по участникам за выбранный период.</EmptyDescription>
    </EmptyHeader>
  </Empty>
  <div v-else class="h-72">
    <Bar :data="chartData" :options="options" />
  </div>
</template>
