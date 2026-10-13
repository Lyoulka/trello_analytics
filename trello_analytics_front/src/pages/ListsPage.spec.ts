import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import ListsPage from './ListsPage.vue'
import { useSessionStore } from '@/stores/session'

vi.mock('@/api/lists', () => ({
  fetchLists: vi.fn(),
  patchList: vi.fn(),
}))

import { fetchLists, patchList } from '@/api/lists'

describe('ListsPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(fetchLists).mockReset()
    vi.mocked(patchList).mockReset()
  })

  it('allows team_lead to toggle is_active', async () => {
    const session = useSessionStore()
    session.user = {
      id: 'u1',
      email: 'lead@example.com',
      display_name: 'Lead',
      role: 'team_lead',
      is_active: true,
    }
    session.ready = true

    vi.mocked(fetchLists).mockResolvedValue({
      lists: [
        {
          id: 'l1',
          trello_list_id: 't1',
          name: 'Icebox',
          is_archived: false,
          is_active: true,
          logical_stage: null,
        },
      ],
    })
    vi.mocked(patchList).mockResolvedValue({
      id: 'l1',
      trello_list_id: 't1',
      name: 'Icebox',
      is_archived: false,
      is_active: false,
      logical_stage: null,
    })

    const wrapper = mount(ListsPage)
    await flushPromises()
    expect(wrapper.text()).toContain('Icebox')
    const btn = wrapper.findAll('button').find((b) => b.text().includes('Исключить'))
    await btn!.trigger('click')
    await flushPromises()
    expect(patchList).toHaveBeenCalledWith('l1', { is_active: false })
    expect(wrapper.text()).toContain('нет')
  })
})
