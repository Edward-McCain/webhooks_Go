export type ApiError = {
  error?: {
    code?: string
    message?: string
    request_id?: string
  }
}

export function useApi() {
  const config = useRuntimeConfig()
  const apiKey = useState<string>('apiKey', () => '')

  async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const headers = new Headers(options.headers || {})
    headers.set('Accept', 'application/json')
    if (options.body && !headers.has('Content-Type')) {
      headers.set('Content-Type', 'application/json')
    }
    if (apiKey.value) {
      headers.set('Authorization', `Bearer ${apiKey.value}`)
    }

    const res = await fetch(`${config.public.apiBase}${path}`, {
      ...options,
      headers,
    })

    if (res.status === 204) {
      return undefined as T
    }

    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      const err = data as ApiError
      throw new Error(err.error?.message || `Request failed (${res.status})`)
    }
    return data as T
  }

  return {
    apiKey,
    get: <T>(path: string) => request<T>(path),
    post: <T>(path: string, body?: unknown) =>
      request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
    patch: <T>(path: string, body?: unknown) =>
      request<T>(path, { method: 'PATCH', body: body ? JSON.stringify(body) : undefined }),
    del: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
  }
}
