import { fetchThroughput } from '@/api/analytics'
import type { PeriodQuery } from '@/types/api'

export interface SeriesPoint {
  label: string
  from: string
  to: string
  /** null when API says unavailable — never fake as 0 */
  count: number | null
}

function addDays(dateOnly: string, days: number): string {
  const d = new Date(`${dateOnly}T00:00:00Z`)
  d.setUTCDate(d.getUTCDate() + days)
  return d.toISOString().slice(0, 10)
}

/**
 * Build daily sub-windows for time-series.
 * `toInclusive` is the last included calendar day (API date-only contract).
 * Each window sends { from: D, to: D } so backend expands once to [D, D+24h).
 * Architect: ≤31 API calls OR Empty — no client aggregation formulas.
 */
export function buildDailyWindows(
  from: string,
  toInclusive: string,
): Array<{ from: string; to: string; label: string }> {
  const windows: Array<{ from: string; to: string; label: string }> = []
  let cursor = from
  while (cursor <= toInclusive) {
    windows.push({ from: cursor, to: cursor, label: cursor })
    cursor = addDays(cursor, 1)
    if (windows.length > 31) break
  }
  return windows
}

export async function fetchThroughputSeries(
  base: PeriodQuery,
  daySpan: number,
): Promise<{ ok: true; points: SeriesPoint[] } | { ok: false; reason: 'too_long' | 'empty' }> {
  if (daySpan <= 0) return { ok: false, reason: 'empty' }
  if (daySpan > 31) return { ok: false, reason: 'too_long' }

  const windows = buildDailyWindows(base.from, base.to)
  if (windows.length === 0) return { ok: false, reason: 'empty' }
  if (windows.length > 31) return { ok: false, reason: 'too_long' }

  const points: SeriesPoint[] = []
  for (const w of windows) {
    const res = await fetchThroughput({
      from: w.from,
      to: w.to,
      member_id: base.member_id,
      label_id: base.label_id,
    })
    points.push({
      label: w.label,
      from: w.from,
      to: w.to,
      count: res.throughput.available ? res.throughput.count : null,
    })
  }
  return { ok: true, points }
}
