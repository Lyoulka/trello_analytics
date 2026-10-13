import { describe, expect, it } from 'vitest'
import { safeInternalPath } from './safeRedirect'

describe('safeInternalPath', () => {
  it('allows relative internal paths', () => {
    expect(safeInternalPath('/dashboard')).toBe('/dashboard')
    expect(safeInternalPath('/cards/abc/history')).toBe('/cards/abc/history')
  })

  it('rejects open redirects', () => {
    expect(safeInternalPath('https://evil.example/phish')).toBe('/dashboard')
    expect(safeInternalPath('//evil.example/phish')).toBe('/dashboard')
    expect(safeInternalPath('javascript:alert(1)')).toBe('/dashboard')
    expect(safeInternalPath('')).toBe('/dashboard')
    expect(safeInternalPath(undefined)).toBe('/dashboard')
  })
})
