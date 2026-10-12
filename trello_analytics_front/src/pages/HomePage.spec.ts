import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import HomePage from './HomePage.vue'

describe('HomePage', () => {
  it('renders product title', () => {
    const wrapper = mount(HomePage)
    expect(wrapper.text()).toContain('Trello Team Analytics')
  })
})
