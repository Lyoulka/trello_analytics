import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useFiltersStore } from './filters'

describe('filters store period contract', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
    // Fixed "today" UTC: 2026-10-13
    vi.setSystemTime(new Date('2026-10-13T15:30:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('14d preset sends N inclusive calendar days ending today (last-included to)', () => {
    const store = useFiltersStore()
    store.applyPreset('14d')
    // Sep 30 .. Oct 13 inclusive = 14 days; to is last included, not exclusive+1
    expect(store.from).toBe('2026-09-30')
    expect(store.to).toBe('2026-10-13')
    expect(store.query).toMatchObject({ from: '2026-09-30', to: '2026-10-13' })
    expect(store.inclusiveDaySpan()).toBe(14)
  })

  it('2d, 7d and 30d presets match inclusive day counts', () => {
    const store = useFiltersStore()
    store.applyPreset('2d')
    expect(store.from).toBe('2026-10-12')
    expect(store.to).toBe('2026-10-13')
    expect(store.inclusiveDaySpan()).toBe(2)

    store.applyPreset('7d')
    expect(store.from).toBe('2026-10-07')
    expect(store.to).toBe('2026-10-13')
    expect(store.inclusiveDaySpan()).toBe(7)

    store.applyPreset('30d')
    expect(store.from).toBe('2026-09-14')
    expect(store.to).toBe('2026-10-13')
    expect(store.inclusiveDaySpan()).toBe(30)
  })

  it('inclusiveDaySpan counts both bounds as included', () => {
    const store = useFiltersStore()
    store.setCustomRange('2026-10-01', '2026-10-01')
    expect(store.inclusiveDaySpan()).toBe(1)

    store.setCustomRange('2026-10-01', '2026-10-03')
    expect(store.inclusiveDaySpan()).toBe(3)
  })
})
