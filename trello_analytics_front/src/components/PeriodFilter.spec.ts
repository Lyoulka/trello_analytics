import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PeriodFilter from './PeriodFilter.vue'
import { useFiltersStore } from '@/stores/filters'

vi.mock('@/api/analytics', () => ({
  fetchWorkload: vi.fn(),
}))

import { fetchWorkload } from '@/api/analytics'

describe('PeriodFilter member control', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(fetchWorkload).mockReset()
    vi.mocked(fetchWorkload).mockResolvedValue({
      workload: [
        {
          member_id: '11111111-1111-1111-1111-111111111111',
          full_name: 'Ada Lovelace',
          active_cards: 1,
          completed_cards: 0,
        },
        {
          member_id: '22222222-2222-2222-2222-222222222222',
          full_name: 'Grace Hopper',
          active_cards: 0,
          completed_cards: 2,
        },
      ],
      warnings: [],
    } as never)
  })

  it('hides member UI by default', () => {
    const wrapper = mount(PeriodFilter)
    expect(wrapper.find('#member-id').exists()).toBe(false)
  })

  it('labels period-to as last-included (включ.), not exclusive', () => {
    const wrapper = mount(PeriodFilter)
    expect(wrapper.find('label[for="period-to"]').text()).toBe('По (включ.)')
  })

  it('shows member names in select and wires filters.memberId when showMember', async () => {
    const wrapper = mount(PeriodFilter, { props: { showMember: true } })
    await flushPromises()

    const select = wrapper.find('#member-id')
    expect(select.exists()).toBe(true)
    expect(fetchWorkload).toHaveBeenCalled()
    const text = select.text()
    expect(text).toContain('Ada Lovelace')
    expect(text).toContain('Grace Hopper')
    expect(text).not.toContain('11111111-1111-1111-1111-111111111111')

    const store = useFiltersStore()
    await select.setValue('11111111-1111-1111-1111-111111111111')
    expect(store.memberId).toBe('11111111-1111-1111-1111-111111111111')
    expect(store.query.member_id).toBe('11111111-1111-1111-1111-111111111111')
  })

  it('clears sticky memberId when member disappears from workload', async () => {
    const store = useFiltersStore()
    store.memberId = '11111111-1111-1111-1111-111111111111'

    vi.mocked(fetchWorkload).mockResolvedValue({
      workload: [
        {
          member_id: '22222222-2222-2222-2222-222222222222',
          full_name: 'Grace Hopper',
          active_cards: 0,
          completed_cards: 2,
        },
      ],
      warnings: [],
    } as never)

    mount(PeriodFilter, { props: { showMember: true } })
    await flushPromises()
    expect(store.memberId).toBe('')
  })
})
