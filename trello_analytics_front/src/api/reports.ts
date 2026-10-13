import type { PeriodQuery, ReportPreview } from '@/types/api'
import { formatMetricCount } from '@/composables/format'
import { apiRequest, periodQueryParams } from './client'

export function fetchReportPreview(q: PeriodQuery) {
  return apiRequest<ReportPreview>('/reports/preview', { query: periodQueryParams(q) })
}

/**
 * Report metrics use JSON null (no separate `available` flag).
 * null/missing → em dash; real 0 stays "0".
 */
export function formatReportMetric(value: number | null | undefined): string {
  return formatMetricCount(value !== null && value !== undefined, value)
}

/** Build plain-text preview from API payload (no client metric formulas). */
export function reportToPlainText(report: ReportPreview): string {
  const lines: string[] = []
  lines.push('Trello Team Analytics — отчёт')
  lines.push(`Период: ${report.period.from} — ${report.period.to}`)
  lines.push('')
  lines.push('Сводка')
  lines.push(`- Завершено: ${formatReportMetric(report.summary.completed_count)}`)
  lines.push(`- Багов исправлено: ${formatReportMetric(report.summary.bugs_fixed_count)}`)
  lines.push(`- Возвратов: ${formatReportMetric(report.summary.reopened_count)}`)
  lines.push(`- В работе (WIP): ${formatReportMetric(report.summary.wip_count)}`)
  if (report.deltas) {
    lines.push('')
    lines.push('Изменение к предыдущему окну')
    lines.push(`- Δ завершённых: ${formatReportMetric(report.deltas.completed_delta)}`)
    lines.push(`- Δ возвратов: ${formatReportMetric(report.deltas.reopened_delta)}`)
  }
  lines.push('')
  lines.push(`Завершённые задачи (${report.completed_tasks.length})`)
  for (const c of report.completed_tasks) {
    const members = (c.member_names ?? []).filter(Boolean).join(', ') || '—'
    lines.push(`- ${c.name} · ${members} · https://trello.com/c/${c.trello_card_id}`)
  }
  lines.push('')
  lines.push(
    `Баги: исправлено=${report.bugs.fixed.length}, снова открыто=${report.bugs.reopened.length}, ещё в работе=${report.bugs.still_in_work.length}`,
  )
  lines.push(
    `Текущая работа: WIP=${report.current_work.wip.length}, зависшие=${report.current_work.stale.length}`,
  )
  if (report.warnings?.length) {
    lines.push('')
    lines.push('Предупреждения')
    for (const w of report.warnings) {
      lines.push(`- [${w.code}] ${w.message}`)
    }
  }
  if (report.problems?.data_quality_hints?.length) {
    lines.push('')
    lines.push('Качество данных')
    for (const h of report.problems.data_quality_hints) {
      lines.push(`- ${h}`)
    }
  }
  return lines.join('\n')
}
