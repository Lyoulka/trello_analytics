import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('./client', () => ({
  apiRequest: vi.fn(),
}))

import { apiRequest } from './client'
import { fetchLists, patchList } from './lists'

describe('lists api', () => {
  beforeEach(() => {
    vi.mocked(apiRequest).mockReset()
  })

  it('fetchLists calls GET /lists', async () => {
    vi.mocked(apiRequest).mockResolvedValue({ lists: [] })
    await fetchLists()
    expect(apiRequest).toHaveBeenCalledWith('/lists')
  })

  it('patchList sends is_active', async () => {
    vi.mocked(apiRequest).mockResolvedValue({
      id: '1',
      trello_list_id: 't1',
      name: 'Icebox',
      is_archived: false,
      is_active: false,
      logical_stage: null,
    })
    await patchList('1', { is_active: false })
    expect(apiRequest).toHaveBeenCalledWith('/lists/1', {
      method: 'PATCH',
      body: { is_active: false },
    })
  })
})
