import type { SyncHistoryResponse, SyncStatusResponse } from '@/types/api'
import { apiRequest } from './client'

export function fetchSyncStatus() {
  return apiRequest<SyncStatusResponse>('/sync/status')
}

export function fetchSyncHistory(limit = 20) {
  return apiRequest<SyncHistoryResponse>('/sync/history', { query: { limit } })
}

export function startSync() {
  return apiRequest<{ run_id: string }>('/sync/start', { method: 'POST' })
}

export function retrySync() {
  return apiRequest<{ run_id: string }>('/sync/retry', { method: 'POST' })
}
