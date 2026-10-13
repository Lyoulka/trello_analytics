import type { OverviewResponse, PeriodQuery, WorkloadEntry, Warning } from '@/types/api'
import { apiRequest, periodQueryParams } from './client'

export function fetchOverview(q: PeriodQuery) {
  return apiRequest<OverviewResponse>('/analytics/overview', { query: periodQueryParams(q) })
}

export function fetchWorkload(q: PeriodQuery) {
  return apiRequest<{ workload: WorkloadEntry[]; warnings: Warning[] }>('/analytics/workload', {
    query: periodQueryParams(q),
  })
}

export function fetchThroughput(q: PeriodQuery) {
  return apiRequest<{ throughput: { available: boolean; count: number }; warnings: Warning[] }>(
    '/analytics/throughput',
    { query: periodQueryParams(q) },
  )
}
