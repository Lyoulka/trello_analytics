import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { formatReportMetric, reportToPlainText } from './reports'
import type { ReportPreview } from '@/types/api'
import KpiCard from '@/components/KpiCard.vue'

function baseReport(overrides: Partial<ReportPreview> = {}): ReportPreview {
  return {
    period: { from: '2026-10-01T00:00:00Z', to: '2026-10-08T00:00:00Z' },
    summary: {
      completed_count: 2,
      bugs_fixed_count: 1,
      reopened_count: 0,
      wip_count: 3,
    },
    completed_tasks: [
      {
        id: '1',
        trello_card_id: 'abc',
        name: 'Task A',
        member_ids: ['m1'],
        member_names: ['Ada'],
        label_ids: [],
        is_bug: false,
      },
    ],
    bugs: { fixed: [], reopened: [], still_in_work: [] },
    current_work: { wip: [], stale: [] },
    problems: {
      stale_card_ids: [],
      reopened_card_ids: [],
      needs_review: [],
      data_quality_hints: ['incomplete history'],
    },
    warnings: [{ code: 'data_stale', message: 'stale sync' }],
    ...overrides,
  }
}

describe('formatReportMetric', () => {
  it('maps null/undefined to em dash and keeps real zero', () => {
    expect(formatReportMetric(null)).toBe('—')
    expect(formatReportMetric(undefined)).toBe('—')
    expect(formatReportMetric(0)).toBe('0')
    expect(formatReportMetric(5)).toBe('5')
  })
})

describe('reportToPlainText', () => {
  it('serializes API preview fields only', () => {
    const text = reportToPlainText(baseReport())
    expect(text).toContain('Task A')
    expect(text).toContain('Ada')
    expect(text).toContain('https://trello.com/c/abc')
    expect(text).toContain('stale sync')
    expect(text).toContain('incomplete history')
    expect(text).toContain('Завершено: 2')
  })

  it('does not print literal null for unavailable summary/deltas (BUG-2)', () => {
    const text = reportToPlainText(
      baseReport({
        summary: {
          completed_count: null,
          bugs_fixed_count: null,
          reopened_count: null,
          wip_count: null,
        },
        deltas: {
          completed_delta: null,
          reopened_delta: null,
        },
      }),
    )
    expect(text).not.toMatch(/\bnull\b/)
    expect(text).toContain('Завершено: —')
    expect(text).toContain('Багов исправлено: —')
    expect(text).toContain('Возвратов: —')
    expect(text).toContain('В работе (WIP): —')
    expect(text).toContain('Δ завершённых: —')
    expect(text).toContain('Δ возвратов: —')
    expect(text).not.toContain('wip_delta')
  })

  it('keeps real zero distinct from unavailable', () => {
    const text = reportToPlainText(
      baseReport({
        summary: {
          completed_count: 0,
          bugs_fixed_count: 0,
          reopened_count: 0,
          wip_count: 0,
        },
      }),
    )
    expect(text).toContain('Завершено: 0')
    expect(text).toContain('В работе (WIP): 0')
    expect(text).not.toContain('Завершено: —')
  })
})

describe('Reports summary KPI honesty (BUG-2)', () => {
  it('renders em dash and «нет данных» for null summary count', () => {
    const wrapper = mount(KpiCard, {
      props: {
        title: 'completed_count',
        value: formatReportMetric(null),
        hint: 'нет данных',
      },
    })
    expect(wrapper.text()).toContain('—')
    expect(wrapper.text()).toContain('нет данных')
    expect(wrapper.text()).not.toMatch(/\b0\b/)
    expect(wrapper.text()).not.toContain('null')
  })
})
