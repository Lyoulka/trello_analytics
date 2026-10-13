/** Display helpers — never coerce null metrics to 0. */

export function formatMetricCount(available: boolean | undefined, count: number | undefined | null): string {
  if (!available || count === null || count === undefined) return '—'
  return String(count)
}

export function formatDurationSeconds(
  available: boolean | undefined,
  seconds: number | undefined | null,
): string {
  if (!available || seconds === null || seconds === undefined) return '—'
  if (seconds < 60) return `${Math.round(seconds)} с`
  if (seconds < 3600) return `${(seconds / 60).toFixed(1)} мин`
  if (seconds < 86400) return `${(seconds / 3600).toFixed(1)} ч`
  return `${(seconds / 86400).toFixed(1)} д`
}

export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString('ru-RU', { timeZone: 'UTC' }) + ' UTC'
}

export function formatPercent(available: boolean | undefined, value: number | undefined | null): string {
  if (!available || value === null || value === undefined) return '—'
  return `${value.toFixed(1)}%`
}
