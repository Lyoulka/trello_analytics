import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import CardHistoryDialog from './CardHistoryDialog.vue'

vi.mock('@/api/cards', () => ({
  fetchCard: vi.fn(),
  fetchCardHistory: vi.fn(),
  trelloCardUrl: (id: string) => `https://trello.com/c/${id}`,
}))

import * as cardsApi from '@/api/cards'

describe('CardHistoryDialog', () => {
  beforeEach(() => {
    vi.mocked(cardsApi.fetchCard).mockReset()
    vi.mocked(cardsApi.fetchCardHistory).mockReset()
  })

  it('loads history when opened with cardId', async () => {
    vi.mocked(cardsApi.fetchCard).mockResolvedValue({
      id: 'id-a',
      trello_card_id: 't-a',
      name: 'Card A',
      current_list_name: 'Doing',
      current_list_id: 'l1',
      is_archived: false,
    } as never)
    vi.mocked(cardsApi.fetchCardHistory).mockResolvedValue({
      events: [
        {
          id: 'e1',
          action_type: 'updateCard',
          action_date: '2026-01-01T00:00:00Z',
          source_list_id: null,
          destination_list_id: null,
          member_id: null,
          source_list_name: 'В работе',
          destination_list_name: 'Готово',
          member_name: 'Ильдар',
        },
      ],
      warnings: [],
    } as never)

    const wrapper = mount(CardHistoryDialog, {
      props: { open: true, cardId: 'id-a' },
      attachTo: document.body,
    })
    await flushPromises()

    expect(cardsApi.fetchCard).toHaveBeenCalledWith('id-a')
    expect(cardsApi.fetchCardHistory).toHaveBeenCalledWith('id-a')
    expect(document.body.textContent).toContain('Card A')
    expect(document.body.textContent).toContain('updateCard')
    expect(document.body.textContent).toContain('В работе → Готово')
    expect(document.body.textContent).toContain('Ильдар')

    wrapper.unmount()
  })

  it('reloads when cardId changes while open', async () => {
    vi.mocked(cardsApi.fetchCard).mockImplementation(async (id: string) =>
      ({
        id,
        trello_card_id: `t-${id}`,
        name: `Card ${id}`,
        current_list_name: 'Doing',
        is_archived: false,
      }) as never,
    )
    vi.mocked(cardsApi.fetchCardHistory).mockResolvedValue({ events: [], warnings: [] } as never)

    const wrapper = mount(CardHistoryDialog, {
      props: { open: true, cardId: 'id-a' },
      attachTo: document.body,
    })
    await flushPromises()
    expect(cardsApi.fetchCard).toHaveBeenCalledWith('id-a')

    await wrapper.setProps({ cardId: 'id-b' })
    await flushPromises()
    expect(cardsApi.fetchCard).toHaveBeenCalledWith('id-b')
    expect(document.body.textContent).toContain('Card id-b')

    wrapper.unmount()
  })
})
