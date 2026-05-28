import type { RememberedAccount } from '@/shared/types/domain'
import { storageKeys } from './keys'

const isRememberedAccount = (value: unknown): value is RememberedAccount => {
  if (!value || typeof value !== 'object') {
    return false
  }

  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.publicId === 'number' &&
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

  getLastUsedPublicId() {
    return window.localStorage.getItem(storageKeys.lastUsedPublicId) ?? ''
  },

  setLastUsedPublicId(publicId: number) {
    window.localStorage.setItem(storageKeys.lastUsedPublicId, String(publicId))
  },

  upsert(account: Omit<RememberedAccount, 'lastLoginAt'>) {
    const next: RememberedAccount = {
      ...account,
      lastLoginAt: new Date().toISOString(),
    }
    const items = readAll().filter((item) => item.publicId !== account.publicId)
    writeAll([next, ...items].sort((a, b) => b.lastLoginAt.localeCompare(a.lastLoginAt)))
    this.setLastUsedPublicId(account.publicId)
  },

  remove(publicId: number) {
    const next = readAll().filter((item) => item.publicId !== publicId)
    writeAll(next)

    if (this.getLastUsedPublicId() === String(publicId)) {
      const fallback = next[0]?.publicId
      if (fallback) {
        this.setLastUsedPublicId(fallback)
      } else {
        window.localStorage.removeItem(storageKeys.lastUsedPublicId)
      }
    }
  },
}
