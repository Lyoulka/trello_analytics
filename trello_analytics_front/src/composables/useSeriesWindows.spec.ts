import { describe, expect, it } from 'vitest'
import { buildDailyWindows } from './useSeriesWindows'

describe('buildDailyWindows', () => {
  it('builds one calendar-day window per inclusive day (date-only to = last included)', () => {
    // from=Oct1, to=Oct3 inclusive → three days; each sends to=same day
    // so backend parseTimeBound expands once to [D, D+24h), not 48h.
    const windows = buildDailyWindows('2026-10-01', '2026-10-03')
    expect(windows).toEqual([
      { from: '2026-10-01', to: '2026-10-01', label: '2026-10-01' },
      { from: '2026-10-02', to: '2026-10-02', label: '2026-10-02' },
      { from: '2026-10-03', to: '2026-10-03', label: '2026-10-03' },
    ])
  })

  it('returns a single window when from === to (one calendar day)', () => {
    expect(buildDailyWindows('2026-10-13', '2026-10-13')).toEqual([
      { from: '2026-10-13', to: '2026-10-13', label: '2026-10-13' },
    ])
  })

  it('returns empty when from is after to', () => {
    expect(buildDailyWindows('2026-10-14', '2026-10-13')).toEqual([])
  })
})
