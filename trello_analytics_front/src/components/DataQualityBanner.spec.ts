import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DataQualityBanner from './DataQualityBanner.vue'

describe('DataQualityBanner', () => {
  it('renders warning messages from API warnings', () => {
    const wrapper = mount(DataQualityBanner, {
      props: {
        warnings: [{ code: 'incomplete_history', message: 'История неполная' }],
      },
    })
    expect(wrapper.text()).toContain('Качество данных')
    expect(wrapper.text()).toContain('История неполная')
  })

  it('renders nothing without messages', () => {
    const wrapper = mount(DataQualityBanner, { props: { warnings: [] } })
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('surfaces primary_import_incomplete from sync status', () => {
    const wrapper = mount(DataQualityBanner, {
      props: {
        sync: {
          board_id: 'b',
          last_success_at: null,
          current_run: null,
          earliest_action_at: null,
          latest_action_at: null,
          counts: { cards: 0, lists: 0, members: 0, labels: 0, actions: 0 },
          primary_import_incomplete: true,
          warnings: ['sparse card history'],
        },
      },
    })
    expect(wrapper.text()).toContain('Первичный импорт')
    expect(wrapper.text()).toContain('sparse card history')
  })
})
