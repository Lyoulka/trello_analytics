import { ApiError, isApiErrorBody } from './errors'

const API_BASE = '/api/v1'

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export interface RequestOptions {
  method?: HttpMethod
  body?: unknown
  query?: Record<string, string | number | boolean | undefined | null>
  /** Skip JSON parse for 204 / empty bodies */
  emptyResponse?: boolean
}

function buildUrl(path: string, query?: RequestOptions['query']): string {
  const base = path.startsWith('http') ? path : `${API_BASE}${path.startsWith('/') ? path : `/${path}`}`
  if (!query) return base
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === '') continue
    params.set(key, String(value))
  }
  const qs = params.toString()
  return qs ? `${base}?${qs}` : base
}

/**
 * Typed fetch client: credentials include (session cookie), envelope errors.
 * Base path `/api/v1` via Vite proxy — no client-side metric formulas.
 */
export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, query, emptyResponse } = options
  const headers: Record<string, string> = {
    Accept: 'application/json',
  }
  let payload: BodyInit | undefined
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }

  const res = await fetch(buildUrl(path, query), {
    method,
    credentials: 'include',
    headers,
    body: payload,
  })

  if (res.status === 204 || emptyResponse) {
    if (!res.ok) {
      throw await toApiError(res)
    }
    return undefined as T
  }

  const text = await res.text()
  let data: unknown = null
  if (text) {
    try {
      data = JSON.parse(text) as unknown
    } catch {
      if (!res.ok) {
        throw new ApiError(res.status, 'invalid_response', 'Сервер вернул некорректный ответ')
      }
      throw new ApiError(res.status, 'invalid_response', 'Не удалось разобрать ответ API')
    }
  }

  if (!res.ok) {
    throw toApiErrorFromBody(res.status, data)
  }

  return data as T
}

async function toApiError(res: Response): Promise<ApiError> {
  try {
    const data = (await res.json()) as unknown
    return toApiErrorFromBody(res.status, data)
  } catch {
    return new ApiError(res.status, 'http_error', `HTTP ${res.status}`)
  }
}

function toApiErrorFromBody(status: number, data: unknown): ApiError {
  if (isApiErrorBody(data)) {
    return new ApiError(status, data.error.code, data.error.message)
  }
  return new ApiError(status, 'http_error', `HTTP ${status}`)
}

export function periodQueryParams(q: {
  from: string
  to: string
  member_id?: string
  label_id?: string
}): Record<string, string> {
  const out: Record<string, string> = { from: q.from, to: q.to }
  if (q.member_id) out.member_id = q.member_id
  if (q.label_id) out.label_id = q.label_id
  return out
}
