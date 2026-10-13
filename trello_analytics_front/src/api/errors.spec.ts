import { describe, expect, it } from 'vitest'
import { ApiError, isApiErrorBody, messageFromUnknown } from './errors'

describe('ApiError envelope', () => {
  it('detects docs/api.md error shape', () => {
    expect(isApiErrorBody({ error: { code: 'forbidden', message: 'nope' } })).toBe(true)
    expect(isApiErrorBody({ message: 'x' })).toBe(false)
  })

  it('flags 401/403 helpers', () => {
    expect(new ApiError(401, 'unauthenticated', 'x').isUnauthorized).toBe(true)
    expect(new ApiError(403, 'forbidden', 'x').isForbidden).toBe(true)
  })

  it('messageFromUnknown prefers ApiError message', () => {
    expect(messageFromUnknown(new ApiError(400, 'bad', 'плохо'))).toBe('плохо')
  })
})
