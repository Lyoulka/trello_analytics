import type { User, UsersListResponse } from '@/types/api'
import { apiRequest } from './client'

export function fetchUsers() {
  return apiRequest<UsersListResponse>('/users')
}

export function createUser(body: {
  email: string
  password: string
  display_name: string
  role: string
}) {
  return apiRequest<User>('/users', { method: 'POST', body })
}

export function patchUser(
  id: string,
  body: {
    display_name?: string
    role?: string
    is_active?: boolean
    password?: string
  },
) {
  return apiRequest<User>(`/users/${id}`, { method: 'PATCH', body })
}
