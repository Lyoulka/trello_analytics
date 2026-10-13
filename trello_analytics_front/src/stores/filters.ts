import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

export type PeriodPreset = '2d' | '7d' | '14d' | '30d' | 'custom'

function toDateOnly(d: Date): string {
  return d.toISOString().slice(0, 10)
}

function startOfUtcDay(d: Date): Date {
  return new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate()))
}

function addUtcDays(d: Date, days: number): Date {
  const next = new Date(d)
  next.setUTCDate(next.getUTCDate() + days)
  return next
}

/**
 * Default: last N inclusive calendar days ending today.
 * Date-only `to` is the last included day — backend parseTimeBound(end=true)
 * expands it to exclusive next midnight UTC (startOfDay(to) + 24h).
 */
function defaultRange(days: number): { from: string; to: string } {
  const today = startOfUtcDay(new Date())
  const from = addUtcDays(today, -(days - 1))
  return { from: toDateOnly(from), to: toDateOnly(today) }
}

export const useFiltersStore = defineStore('filters', () => {
  const initial = defaultRange(14)
  const from = ref(initial.from)
  const to = ref(initial.to)
  const preset = ref<PeriodPreset>('14d')
  const memberId = ref<string>('')
  const labelId = ref<string>('')

  const query = computed(() => ({
    from: from.value,
    to: to.value,
    member_id: memberId.value || undefined,
    label_id: labelId.value || undefined,
  }))

  function applyPreset(p: PeriodPreset) {
    preset.value = p
    if (p === 'custom') return
    const days = p === '2d' ? 2 : p === '7d' ? 7 : p === '14d' ? 14 : 30
    const range = defaultRange(days)
    from.value = range.from
    to.value = range.to
  }

  function setCustomRange(nextFrom: string, nextTo: string) {
    preset.value = 'custom'
    from.value = nextFrom
    to.value = nextTo
  }

  /** Inclusive calendar day count for series window checks (≤31). */
  function inclusiveDaySpan(): number {
    const a = Date.parse(`${from.value}T00:00:00Z`)
    const b = Date.parse(`${to.value}T00:00:00Z`)
    if (Number.isNaN(a) || Number.isNaN(b) || b < a) return 0
    // Both bounds are inclusive calendar days.
    return Math.round((b - a) / 86_400_000) + 1
  }

  return {
    from,
    to,
    preset,
    memberId,
    labelId,
    query,
    applyPreset,
    setCustomRange,
    inclusiveDaySpan,
  }
})
