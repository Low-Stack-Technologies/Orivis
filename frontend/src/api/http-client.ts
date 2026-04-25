import { getAccessToken } from '../auth/session'

export async function customFetch<T>(config: {
  url: string
  method: string
  params?: Record<string, unknown>
  data?: unknown
  headers?: Record<string, string>
  signal?: AbortSignal
}): Promise<T> {
  const url = new URL(config.url, window.location.origin)
  const accessToken = getAccessToken()

  if (config.params) {
    for (const [key, value] of Object.entries(config.params)) {
      if (value !== undefined && value !== null) {
        url.searchParams.set(key, String(value))
      }
    }
  }

  const bodyIsForm =
    config.headers?.['Content-Type'] === 'application/x-www-form-urlencoded' &&
    config.data &&
    typeof config.data === 'object'

  const response = await fetch(url.toString(), {
    method: config.method,
    signal: config.signal,
    headers: {
      'Content-Type': bodyIsForm ? 'application/x-www-form-urlencoded' : 'application/json',
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      ...(config.headers || {})
    },
    credentials: 'include',
    body: !config.data
      ? undefined
      : bodyIsForm
        ? new URLSearchParams(config.data as Record<string, string>).toString()
        : JSON.stringify(config.data)
  })

  if (!response.ok) {
    let message = `Request failed with status ${response.status}`

    try {
      const errorPayload = (await response.json()) as { message?: string }
      if (errorPayload?.message) {
        message = errorPayload.message
      }
    } catch {
      // ignore parse failures and keep fallback message
    }

    throw new Error(message)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return (await response.json()) as T
}
