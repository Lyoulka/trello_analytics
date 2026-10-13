/** Parse required non-negative integer from form text; reject empty/NaN without silent defaults. */
export function parseNonNegInt(raw: string, fieldLabel: string): number {
  const trimmed = raw.trim()
  if (trimmed === '') {
    throw new Error(`${fieldLabel}: значение обязательно`)
  }
  const n = Number(trimmed)
  if (!Number.isFinite(n) || !Number.isInteger(n) || n < 0) {
    throw new Error(`${fieldLabel}: ожидается целое число ≥ 0`)
  }
  return n
}
