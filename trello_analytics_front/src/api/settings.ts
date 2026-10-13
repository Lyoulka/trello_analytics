import type { AppSettingsResponse, WorkflowSettings } from '@/types/api'
import { apiRequest } from './client'

export function fetchSettings() {
  return apiRequest<AppSettingsResponse>('/settings')
}

export function patchSettings(settings: Record<string, string>) {
  return apiRequest<AppSettingsResponse>('/settings', {
    method: 'PATCH',
    body: { settings },
  })
}

export function fetchWorkflow() {
  return apiRequest<WorkflowSettings>('/settings/workflow')
}

export function putWorkflow(body: WorkflowSettings) {
  return apiRequest<WorkflowSettings>('/settings/workflow', {
    method: 'PUT',
    body,
  })
}
