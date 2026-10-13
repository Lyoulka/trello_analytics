import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as authApi from '@/api/auth'
import { ApiError } from '@/api/errors'
import type { User } from '@/types/api'
import { roleAtLeast, type AppRole } from '@/types/roles'

export const useSessionStore = defineStore('session', () => {
  const user = ref<User | null>(null)
  const ready = ref(false)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isAuthenticated = computed(() => !!user.value)
  const role = computed(() => user.value?.role ?? null)

  function hasMinRole(min: AppRole): boolean {
    return roleAtLeast(role.value, min)
  }

  async function refreshMe(): Promise<boolean> {
    loading.value = true
    error.value = null
    try {
      user.value = await authApi.fetchMe()
      return true
    } catch (err) {
      user.value = null
      if (err instanceof ApiError && err.isUnauthorized) {
        return false
      }
      error.value = err instanceof Error ? err.message : 'Не удалось проверить сессию'
      return false
    } finally {
      loading.value = false
      ready.value = true
    }
  }

  async function login(email: string, password: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      user.value = await authApi.login(email, password)
      ready.value = true
    } catch (err) {
      user.value = null
      throw err
    } finally {
      loading.value = false
    }
  }

  async function logout(): Promise<void> {
    try {
      await authApi.logout()
    } finally {
      user.value = null
    }
  }

  return {
    user,
    ready,
    loading,
    error,
    isAuthenticated,
    role,
    hasMinRole,
    refreshMe,
    login,
    logout,
  }
})
