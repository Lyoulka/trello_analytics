import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import LoginPage from './LoginPage.vue'

vi.mock('@/api/auth', () => ({
  login: vi.fn(),
  logout: vi.fn(),
  fetchMe: vi.fn(),
}))

import * as authApi from '@/api/auth'

describe('LoginPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(authApi.login).mockReset()
  })

  it('shows error state on failed login', async () => {
    vi.mocked(authApi.login).mockRejectedValue(new Error('invalid email or password'))
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/login', component: LoginPage },
        { path: '/dashboard', component: { template: '<div>dash</div>' } },
      ],
    })
    await router.push('/login')
    await router.isReady()

    const wrapper = mount(LoginPage, {
      global: { plugins: [pinia, router] },
    })

    await wrapper.find('#email').setValue('a@b.c')
    await wrapper.find('#password').setValue('wrong-password')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.text()).toContain('Ошибка входа')
    expect(wrapper.text()).toContain('invalid email or password')
  })

  it('rejects open-redirect query and goes to /dashboard', async () => {
    vi.mocked(authApi.login).mockResolvedValue({
      id: 'u1',
      email: 'a@b.c',
      display_name: 'A',
      role: 'member',
      active: true,
    } as never)

    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/login', component: LoginPage },
        { path: '/dashboard', name: 'dashboard', component: { template: '<div>dash</div>' } },
      ],
    })
    await router.push({ path: '/login', query: { redirect: '//evil.example/x' } })
    await router.isReady()

    const wrapper = mount(LoginPage, {
      global: { plugins: [pinia, router] },
    })

    await wrapper.find('#email').setValue('a@b.c')
    await wrapper.find('#password').setValue('ok')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/dashboard')
  })
})
