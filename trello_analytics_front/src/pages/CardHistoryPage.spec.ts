import { describe, expect, it } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import CardHistoryPage from './CardHistoryPage.vue'

describe('CardHistoryPage', () => {
  it('redirects /cards/:id/history to /cards?history=:id', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/cards/:id/history', name: 'card-history', component: CardHistoryPage },
        { path: '/cards', name: 'cards', component: { template: '<div>cards</div>' } },
      ],
    })
    await router.push('/cards/id-a/history')
    await router.isReady()

    mount(CardHistoryPage, {
      global: { plugins: [router] },
    })
    await flushPromises()

    expect(router.currentRoute.value.name).toBe('cards')
    expect(router.currentRoute.value.query.history).toBe('id-a')
  })
})
