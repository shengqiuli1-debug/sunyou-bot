const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1'

export class ApiError extends Error {
  status: number
  data: any

  constructor(message: string, status: number, data: any) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.data = data
  }
}

function getToken() {
  return localStorage.getItem('sunyou_token') || ''
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(init.headers as Record<string, string> | undefined)
  }
  const token = getToken()
  if (token) headers['X-User-Token'] = token

  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new ApiError(data.error || `Request failed (${res.status})`, res.status, data)
  }
  return data as T
}

export const http = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined })
}
