import { describe, expect, it } from 'vitest'
import { formatDurationSeconds, formatMetricCount, formatPercent } from './format'

describe('formatMetricCount', () => {
  it('shows em dash when unavailable (null/missing ≠ 0)', () => {
    expect(formatMetricCount(false, 0)).toBe('—')
    expect(formatMetricCount(false, undefined)).toBe('—')
    expect(formatMetricCount(true, null)).toBe('—')
  })

  it('shows real zero only when available', () => {
    expect(formatMetricCount(true, 0)).toBe('0')
    expect(formatMetricCount(true, 12)).toBe('12')
  })
})

describe('formatDurationSeconds', () => {
  it('does not fake missing durations as zero', () => {
    expect(formatDurationSeconds(false, 0)).toBe('—')
    expect(formatDurationSeconds(true, undefined)).toBe('—')
  })
})

describe('formatPercent', () => {
  it('keeps unavailable distinct from 0%', () => {
    expect(formatPercent(false, 0)).toBe('—')
    expect(formatPercent(true, 0)).toBe('0.0%')
  })
})
