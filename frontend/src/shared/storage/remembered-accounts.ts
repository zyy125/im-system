import type { RememberedAccount } from '@/shared/types/domain'
import { storageKeys } from './keys'

const isRememberedAccount = (value: unknown): value is RememberedAccount => {
  if (!value || typeof value !== 'object') {
    return false
  }

  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.username === 'string' &&
    typeof candidate.lastLoginAt === 'string'
  )
}

const readAll = () => {
  const raw = window.localStorage.getItem(storageKeys.rememberedAccounts)
  if (!raw) {
    return [] as RememberedAccount[]
  }

  try {
    const parsed = JSON.parse(raw) as unknown[]
    return parsed.filter(isRememberedAccount).sort((a, b) =>
      b.lastLoginAt.localeCompare(a.lastLoginAt),
    )
  } catch {
    window.localStorage.removeItem(storageKeys.rememberedAccounts)
    return [] as RememberedAccount[]
  }
}

const writeAll = (accounts: RememberedAccount[]) => {
  window.localStorage.setItem(storageKeys.rememberedAccounts, JSON.stringify(accounts))
}

export const rememberedAccountsStorage = {
  list: readAll,

  getLastUsedUsername() {
    return window.localStorage.getItem(storageKeys.lastUsedUsername) ?? ''
  },

  setLastUsedUsername(username: string) {
    window.localStorage.setItem(storageKeys.lastUsedUsername, username)
  },

  upsert(account: Omit<RememberedAccount, 'lastLoginAt'>) {
    const next: RememberedAccount = {
      ...account,
      lastLoginAt: new Date().toISOString(),
    }
    const items = readAll().filter((item) => item.username !== account.username)
    writeAll([next, ...items].sort((a, b) => b.lastLoginAt.localeCompare(a.lastLoginAt)))
    this.setLastUsedUsername(account.username)
  },

  remove(username: string) {
    const next = readAll().filter((item) => item.username !== username)
    writeAll(next)

    if (this.getLastUsedUsername() === username) {
      const fallback = next[0]?.username
      if (fallback) {
        this.setLastUsedUsername(fallback)
      } else {
        window.localStorage.removeItem(storageKeys.lastUsedUsername)
      }
    }
  },
}
