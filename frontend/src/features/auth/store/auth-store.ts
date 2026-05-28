import { create } from 'zustand'
import { rememberedAccountsStorage } from '@/shared/storage/remembered-accounts'
import { tokenStorage } from '@/shared/storage/tokens'
import type { AuthTokens, CurrentUser, RememberedAccount } from '@/shared/types/domain'

interface AuthState {
  tokens: AuthTokens | null
  currentUser: CurrentUser | null
  rememberedAccounts: RememberedAccount[]
  bootstrapped: boolean
  hydrate: () => void
  setCurrentUser: (user: CurrentUser | null) => void
  completeLogin: (payload: { tokens: AuthTokens; user: CurrentUser }) => void
  removeRememberedAccount: (publicId: number) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  tokens: null,
  currentUser: null,
  rememberedAccounts: [],
  bootstrapped: false,

  hydrate: () =>
    set({
      tokens: tokenStorage.get(),
      rememberedAccounts: rememberedAccountsStorage.list(),
      bootstrapped: true,
    }),

  setCurrentUser: (currentUser) => set({ currentUser }),

  completeLogin: ({ tokens, user }) => {
    tokenStorage.set(tokens)
    rememberedAccountsStorage.upsert({
      publicId: user.publicId,
      username: user.username,
    })

    set({
      tokens,
      currentUser: user,
      rememberedAccounts: rememberedAccountsStorage.list(),
    })
  },

  removeRememberedAccount: (publicId) => {
    rememberedAccountsStorage.remove(publicId)
    set({ rememberedAccounts: rememberedAccountsStorage.list() })
  },

  logout: () => {
    tokenStorage.clear()
    set({
      tokens: null,
      currentUser: null,
      rememberedAccounts: rememberedAccountsStorage.list(),
    })
  },
}))
