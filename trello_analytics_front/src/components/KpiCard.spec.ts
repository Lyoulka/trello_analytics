import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import KpiCard from './KpiCard.vue'

describe('KpiCard', () => {
  it('displays em dash for missing metrics without inventing zero', () => {
    const wrapper = mount(KpiCard, {
      props: { title: 'Пропускная способность', value: '—', hint: 'нет данных' },
    })
    expect(wrapper.text()).toContain('—')
    expect(wrapper.text()).toContain('нет данных')
    expect(wrapper.text()).not.toMatch(/\b0\b/)
  })
})
