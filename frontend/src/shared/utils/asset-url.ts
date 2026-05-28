import { env } from '@/shared/utils/env'

const ABSOLUTE_URL_PATTERN = /^(?:[a-z]+:)?\/\//i

export function resolveAssetUrl(value?: string | null) {
  const trimmed = value?.trim()
  if (!trimmed) {
    return ''
  }

  if (
    ABSOLUTE_URL_PATTERN.test(trimmed) ||
    trimmed.startsWith('data:') ||
    trimmed.startsWith('blob:')
  ) {
    return trimmed
  }

  if (trimmed.startsWith('/')) {
    return `${env.apiBaseUrl}${trimmed}`
  }

  return `${env.apiBaseUrl}/${trimmed.replace(/^\/+/, '')}`
}
