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
  public_id: number
  username: string
  online: boolean
}): CurrentUser => ({
  publicId: payload.public_id,
  username: payload.username,
  online: payload.online,
})

export const useAuthActions = () => {
  const completeLogin = useAuthStore((state) => state.completeLogin)
  const setCurrentUser = useAuthStore((state) => state.setCurrentUser)
  const logoutState = useAuthStore((state) => state.logout)

  const login = async (publicId: number, password: string) => {
    const loginResult = await authApi.login({
      public_id: publicId,
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
      const registerResult = await authApi.register({ username, password })
      await login(registerResult.public_id, password)
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
