import type { BoardMember, MembersListResponse } from '@/types/api'
import { apiRequest } from './client'

export function fetchMembers() {
  return apiRequest<MembersListResponse>('/members')
}

export function patchMember(id: string, body: { is_active: boolean }) {
  return apiRequest<BoardMember>(`/members/${id}`, { method: 'PATCH', body })
}
