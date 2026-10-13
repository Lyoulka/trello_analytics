export type AppRole = 'admin' | 'team_lead' | 'member'

const ROLE_RANK: Record<AppRole, number> = {
  member: 1,
  team_lead: 2,
  admin: 3,
}

export function isAppRole(value: string): value is AppRole {
  return value === 'admin' || value === 'team_lead' || value === 'member'
}

/** Hierarchy Admin ⊃ Team Lead ⊃ Member (UI hide ≠ security). */
export function roleAtLeast(role: string | null | undefined, min: AppRole): boolean {
  if (!role || !isAppRole(role)) return false
  return ROLE_RANK[role] >= ROLE_RANK[min]
}
