import { describe, expect, it } from 'vitest'
import { chartCssColor } from './chartColors'

describe('chartCssColor', () => {
  it('reads --chart-* from document CSS variables', () => {
    document.documentElement.style.setProperty('--chart-1', 'oklch(0.5 0.1 100)')
    expect(chartCssColor(1)).toBe('oklch(0.5 0.1 100)')
    document.documentElement.style.removeProperty('--chart-1')
  })

  it('falls back when token is missing', () => {
    document.documentElement.style.removeProperty('--chart-2')
    expect(chartCssColor(2)).toBe('#64748b')
  })
})
