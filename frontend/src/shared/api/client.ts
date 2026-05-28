import { env } from '@/shared/utils/env'
import { ApiError } from '@/shared/types/api'
import type { ApiEnvelope, ApiErrorPayload } from '@/shared/types/api'
import { tokenStorage } from '@/shared/storage/tokens'

interface RequestOptions extends RequestInit {
  auth?: boolean
  accessToken?: string
  _retried?: boolean
}

async function parsePayload<T>(response: Response) {
  return (await response.json()) as ApiEnvelope<T> | ApiErrorPayload
}

async function tryRefreshTokens() {
  const tokens = tokenStorage.get()
  if (!tokens?.refreshToken) {
    return null
  }

  const response = await fetch(`${env.apiBaseUrl}/api/v1/auth/refresh`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      refresh_token: tokens.refreshToken,
    }),
  })

  const payload = await parsePayload<{
    access_token: string
    refresh_token: string
    expires_in: number
  }>(response)

  if (!response.ok || payload.code !== 'ok' || !payload.data) {
    tokenStorage.clear()
    return null
  }

  const nextTokens = {
    accessToken: payload.data.access_token,
    refreshToken: payload.data.refresh_token,
    expiresIn: payload.data.expires_in,
  }
  tokenStorage.set(nextTokens)
  return nextTokens
}

export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T | null> {
  const headers = new Headers(options.headers)

  if (options.body && !(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  if (options.auth) {
    const token = options.accessToken || tokenStorage.get()?.accessToken
    if (token) {
      headers.set('Authorization', `Bearer ${token}`)
    }
  }

  const response = await fetch(`${env.apiBaseUrl}${path}`, {
    ...options,
    headers,
  })

  const payload = await parsePayload<T>(response)

  if (
    options.auth &&
    !options._retried &&
    response.status === 401 &&
    payload.code !== 'auth.refresh_token_invalid' &&
    payload.code !== 'auth.token_missing'
  ) {
    const refreshed = await tryRefreshTokens()
    if (refreshed) {
      return apiRequest<T>(path, {
        ...options,
        accessToken: refreshed.accessToken,
        _retried: true,
      })
    }
  }

  if (!response.ok || payload.code !== 'ok') {
    throw new ApiError({
      code: payload.code,
      message: payload.message || 'Request failed',
      status: response.status,
    })
  }

  return payload.data
}

export async function apiRequestOrThrow<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const data = await apiRequest<T>(path, options)

  if (data === null) {
    throw new ApiError({
      code: 'common.invalid_response',
      message: 'Response data is empty',
      status: 500,
    })
  }

  return data
}
