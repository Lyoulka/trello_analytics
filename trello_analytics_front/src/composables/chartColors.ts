/** Read Chart.js colors from CSS design tokens (`--chart-*`), not hardcoded duplicates. */

export type ChartToken = 1 | 2 | 3 | 4 | 5

/** Neutral fallback only when CSS tokens are unavailable (SSR / missing theme). */
const FALLBACK = '#64748b'

export function chartCssColor(token: ChartToken): string {
  if (typeof document === 'undefined') return FALLBACK
  const raw = getComputedStyle(document.documentElement)
    .getPropertyValue(`--chart-${token}`)
    .trim()
  return raw || FALLBACK
}
