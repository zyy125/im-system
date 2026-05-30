const trimTrailingSlash = (value: string) => value.replace(/\/+$/, '')

const defaultApiBaseUrl = import.meta.env.DEV ? 'http://localhost:8080' : ''
const rawApiBaseUrl = import.meta.env.VITE_API_BASE_URL?.trim() ?? defaultApiBaseUrl

export const env = {
  apiBaseUrl: rawApiBaseUrl ? trimTrailingSlash(rawApiBaseUrl) : '',
  wsBaseUrl: rawApiBaseUrl
    ? trimTrailingSlash(rawApiBaseUrl).replace(/^http/i, 'ws')
    : window.location.origin.replace(/^http/i, 'ws'),
}
