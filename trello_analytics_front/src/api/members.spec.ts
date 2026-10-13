import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('./client', () => ({
  apiRequest: vi.fn(),
}))

import { apiRequest } from './client'
import { fetchMembers, patchMember } from './members'

describe('members api', () => {
  beforeEach(() => {
    vi.mocked(apiRequest).mockReset()
  })

  it('fetchMembers calls GET /members', async () => {
    vi.mocked(apiRequest).mockResolvedValue({ members: [] })
    await fetchMembers()
    expect(apiRequest).toHaveBeenCalledWith('/members')
  })

  it('patchMember sends is_active', async () => {
    vi.mocked(apiRequest).mockResolvedValue({
      id: '1',
      trello_member_id: 't1',
      full_name: 'A',
      username: 'a',
      is_active: false,
    })
    await patchMember('1', { is_active: false })
    expect(apiRequest).toHaveBeenCalledWith('/members/1', {
      method: 'PATCH',
      body: { is_active: false },
    })
  })
})
