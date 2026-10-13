import type { BoardList, ListsListResponse } from '@/types/api'
import { apiRequest } from './client'

export function fetchLists() {
  return apiRequest<ListsListResponse>('/lists')
}

export function patchList(id: string, body: { is_active: boolean }) {
  return apiRequest<BoardList>(`/lists/${id}`, { method: 'PATCH', body })
}
