import { env } from '@/shared/utils/env'
import { ApiError } from '@/shared/types/api'
import type { ApiEnvelope, ApiErrorPayload } from '@/shared/types/api'
import { tokenStorage } from '@/shared/storage/tokens'

interface RequestOptions extends RequestInit {
  auth?: boolean
  accessToken?: string
}

export async function apiRequest<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T | null> {
  const headers = new Headers(options.headers)

  if (options.body && !headers.has('Content-Type')) {
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

  const payload = (await response.json()) as ApiEnvelope<T> | ApiErrorPayload

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
