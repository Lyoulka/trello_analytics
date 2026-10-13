import type { CardDetail, CardHistoryResponse, CardsListResponse } from '@/types/api'
import { apiRequest } from './client'

export interface CardsListQuery {
  limit?: number
  offset?: number
  sort?: 'name' | 'updated' | 'created'
  order?: 'asc' | 'desc'
  member_id?: string
  label_id?: string
  list_id?: string
  archived?: boolean
  q?: string
  /** Inclusive calendar day YYYY-MM-DD (or RFC3339). */
  updated_from?: string
  /** Inclusive last calendar day YYYY-MM-DD (backend expands to exclusive next midnight). */
  updated_to?: string
}

export function fetchCards(query: CardsListQuery = {}) {
  return apiRequest<CardsListResponse>('/cards', {
    query: {
      limit: query.limit,
      offset: query.offset,
      sort: query.sort,
      order: query.order,
      member_id: query.member_id,
      label_id: query.label_id,
      list_id: query.list_id,
      archived: query.archived === undefined ? undefined : query.archived ? 'true' : 'false',
      q: query.q,
      updated_from: query.updated_from,
      updated_to: query.updated_to,
    },
  })
}

export function fetchCard(id: string) {
  return apiRequest<CardDetail>(`/cards/${id}`)
}

export function fetchCardHistory(id: string) {
  return apiRequest<CardHistoryResponse>(`/cards/${id}/history`)
}

/** Trello deep-link from API field only — never invent card ids. */
export function trelloCardUrl(trelloCardId: string): string {
  return `https://trello.com/c/${trelloCardId}`
}
