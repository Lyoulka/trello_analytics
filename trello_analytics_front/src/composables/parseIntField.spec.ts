import { describe, expect, it } from 'vitest'
import { parseNonNegInt } from './parseIntField'

describe('parseNonNegInt', () => {
  it('accepts zero and positive integers', () => {
    expect(parseNonNegInt('0', 'x')).toBe(0)
    expect(parseNonNegInt('604800', 'x')).toBe(604800)
  })

  it('rejects empty and invalid without silent defaults', () => {
    expect(() => parseNonNegInt('', 'field')).toThrow(/обязательно/)
    expect(() => parseNonNegInt('abc', 'field')).toThrow(/целое число/)
    expect(() => parseNonNegInt('-1', 'field')).toThrow(/целое число/)
    expect(() => parseNonNegInt('1.5', 'field')).toThrow(/целое число/)
  })
})
