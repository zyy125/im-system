import { authApi } from '@/features/auth/api/auth-api'
import { useAuthStore } from '@/features/auth/store/auth-store'
import type { AuthTokens, CurrentUser } from '@/shared/types/domain'

const mapTokens = (payload: {
  access_token: string
  refresh_token: string
  expires_in: number
}): AuthTokens => ({
  accessToken: payload.access_token,
  refreshToken: payload.refresh_token,
  expiresIn: payload.expires_in,
})

const mapCurrentUser = (payload: {
  user_id: number
  avatar: string
  username: string
  online: boolean
}): CurrentUser => ({
  userId: payload.user_id,
  avatar: payload.avatar,
  username: payload.username,
  online: payload.online,
})

export const useAuthActions = () => {
  const completeLogin = useAuthStore((state) => state.completeLogin)
  const setCurrentUser = useAuthStore((state) => state.setCurrentUser)
  const logoutState = useAuthStore((state) => state.logout)

  const login = async (username: string, password: string) => {
    const loginResult = await authApi.login({
      username,
      password,
    })
    const tokens = mapTokens(loginResult)
    const me = await authApi.getMe(tokens.accessToken)

    completeLogin({
      tokens,
      user: mapCurrentUser(me),
    })
  }

  return {
    login,

    async register(username: string, password: string) {
      await authApi.register({ username, password })
      await login(username, password)
    },

    async logout() {
      try {
        await authApi.logout()
      } catch {
        // We still clear local auth state if server logout fails.
      } finally {
        logoutState()
      }
    },

    async restoreSession() {
      const tokens = useAuthStore.getState().tokens
      if (!tokens?.accessToken) {
        return
      }

      const me = await authApi.getMe(tokens.accessToken)
      setCurrentUser(mapCurrentUser(me))
    },

    mapCurrentUser,
  }
}
