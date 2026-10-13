import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import CardsPage from './CardsPage.vue'

vi.mock('@/api/cards', () => ({
  fetchCards: vi.fn(),
  fetchCard: vi.fn(),
  fetchCardHistory: vi.fn(),
  trelloCardUrl: (id: string) => `https://trello.com/c/${id}`,
}))

vi.mock('@/api/members', () => ({
  fetchMembers: vi.fn(),
}))

import * as cardsApi from '@/api/cards'
import { fetchMembers } from '@/api/members'

describe('CardsPage history modal', () => {
  beforeEach(() => {
    vi.mocked(cardsApi.fetchCards).mockReset()
    vi.mocked(cardsApi.fetchCard).mockReset()
    vi.mocked(cardsApi.fetchCardHistory).mockReset()
    vi.mocked(fetchMembers).mockReset()
    vi.mocked(cardsApi.fetchCards).mockResolvedValue({
      items: [
        {
          id: 'c1',
          trello_card_id: 't1',
          name: 'Task One',
          current_list_name: 'Doing',
          last_trello_update_at: null,
          last_updated_by_name: null,
          is_archived: false,
          member_ids: ['m1'],
          label_ids: [],
        },
      ],
      total: 1,
      limit: 25,
      offset: 0,
    } as never)
    vi.mocked(cardsApi.fetchCard).mockResolvedValue({
      id: 'c1',
      trello_card_id: 't1',
      name: 'Task One',
      current_list_name: 'Doing',
      is_archived: false,
    } as never)
    vi.mocked(cardsApi.fetchCardHistory).mockResolvedValue({
      events: [],
      warnings: [],
    } as never)
    vi.mocked(fetchMembers).mockResolvedValue({
      members: [
        {
          id: 'm1',
          trello_member_id: 'tm1',
          full_name: 'Ada Lovelace',
          username: 'ada',
          is_active: true,
        },
      ],
    })
  })

  it('opens history dialog without leaving /cards', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/cards', name: 'cards', component: CardsPage }],
    })
    await router.push('/cards')
    await router.isReady()

    const wrapper = mount(CardsPage, {
      global: { plugins: [router] },
      attachTo: document.body,
    })
    await flushPromises()

    const historyBtn = wrapper.findAll('button').find((b) => b.text().includes('история'))
    expect(historyBtn).toBeTruthy()
    await historyBtn!.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/cards')
    expect(router.currentRoute.value.query.history).toBe('c1')
    expect(cardsApi.fetchCardHistory).toHaveBeenCalledWith('c1')
    expect(document.body.textContent).toContain('Task One')

    wrapper.unmount()
  })

  it('shows assignees resolved from members', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/cards', name: 'cards', component: CardsPage }],
    })
    await router.push('/cards')
    await router.isReady()

    const wrapper = mount(CardsPage, {
      global: { plugins: [router] },
      attachTo: document.body,
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Исполнители')
    expect(wrapper.text()).toContain('Ada Lovelace')
    expect(fetchMembers).toHaveBeenCalled()

    wrapper.unmount()
  })
})
