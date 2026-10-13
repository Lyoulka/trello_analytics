import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import MembersPage from './MembersPage.vue'
import { useSessionStore } from '@/stores/session'

vi.mock('@/api/members', () => ({
  fetchMembers: vi.fn(),
  patchMember: vi.fn(),
}))

import { fetchMembers, patchMember } from '@/api/members'

describe('MembersPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(fetchMembers).mockReset()
    vi.mocked(patchMember).mockReset()
  })

  it('renders members and allows team_lead to toggle is_active', async () => {
    const session = useSessionStore()
    session.user = {
      id: 'u1',
      email: 'lead@example.com',
      display_name: 'Lead',
      role: 'team_lead',
      is_active: true,
    }
    session.ready = true

    vi.mocked(fetchMembers).mockResolvedValue({
      members: [
        {
          id: 'm1',
          trello_member_id: 't1',
          full_name: 'Илья',
          username: 'ilya',
          is_active: true,
        },
      ],
    })
    vi.mocked(patchMember).mockResolvedValue({
      id: 'm1',
      trello_member_id: 't1',
      full_name: 'Илья',
      username: 'ilya',
      is_active: false,
    })

    const wrapper = mount(MembersPage)
    await flushPromises()

    expect(wrapper.text()).toContain('Илья')
    const btn = wrapper.findAll('button').find((b) => b.text().includes('Исключить'))
    expect(btn).toBeTruthy()
    await btn!.trigger('click')
    await flushPromises()

    expect(patchMember).toHaveBeenCalledWith('m1', { is_active: false })
    expect(wrapper.text()).toContain('нет')
  })

  it('hides toggle for member role', async () => {
    const session = useSessionStore()
    session.user = {
      id: 'u2',
      email: 'm@example.com',
      display_name: 'Mem',
      role: 'member',
      is_active: true,
    }
    session.ready = true

    vi.mocked(fetchMembers).mockResolvedValue({
      members: [
        {
          id: 'm1',
          trello_member_id: 't1',
          full_name: 'Илья',
          username: 'ilya',
          is_active: true,
        },
      ],
    })

    const wrapper = mount(MembersPage)
    await flushPromises()
    expect(wrapper.text()).toContain('Илья')
    expect(wrapper.text()).not.toContain('Исключить')
  })
})
