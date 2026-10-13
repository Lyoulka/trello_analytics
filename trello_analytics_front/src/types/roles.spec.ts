import { describe, expect, it } from 'vitest'
import { roleAtLeast } from './roles'

describe('roleAtLeast', () => {
  it('respects Admin ⊃ Team Lead ⊃ Member', () => {
    expect(roleAtLeast('member', 'member')).toBe(true)
    expect(roleAtLeast('member', 'team_lead')).toBe(false)
    expect(roleAtLeast('team_lead', 'member')).toBe(true)
    expect(roleAtLeast('admin', 'team_lead')).toBe(true)
    expect(roleAtLeast(null, 'member')).toBe(false)
  })
})
