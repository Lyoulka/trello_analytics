/**
 * Allow only internal relative paths: start with `/`, reject protocol-relative `//`.
 * Prevents open-redirect via `?redirect=https://…` or `//evil`.
 */
export function safeInternalPath(raw: unknown, fallback = '/dashboard'): string {
  if (typeof raw !== 'string' || raw.length === 0) return fallback
  if (!raw.startsWith('/') || raw.startsWith('//')) return fallback
  return raw
}
