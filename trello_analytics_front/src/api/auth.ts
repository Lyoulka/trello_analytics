import type { User } from '@/types/api'
import { apiRequest } from './client'

export function login(email: string, password: string) {
  return apiRequest<User>('/auth/login', {
    method: 'POST',
    body: { email, password },
  })
}

export function logout() {
  return apiRequest<void>('/auth/logout', { method: 'POST', emptyResponse: true })
}

export function fetchMe() {
  return apiRequest<User>('/auth/me')
}
