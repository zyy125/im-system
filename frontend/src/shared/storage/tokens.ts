import type { AuthTokens } from '@/shared/types/domain'
import { storageKeys } from './keys'

export const tokenStorage = {
  get() {
    const raw = window.localStorage.getItem(storageKeys.tokens)
    if (!raw) {
      return null
    }

    try {
      return JSON.parse(raw) as AuthTokens
    } catch {
      window.localStorage.removeItem(storageKeys.tokens)
      return null
    }
  },

  set(tokens: AuthTokens) {
    window.localStorage.setItem(storageKeys.tokens, JSON.stringify(tokens))
  },

  clear() {
    window.localStorage.removeItem(storageKeys.tokens)
  },
}
