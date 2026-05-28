import { apiRequest, apiRequestOrThrow } from '@/shared/api/client'
import type {
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RegisterResponse,
  UserProfileResponse,
} from '@/features/auth/types/auth'

export const authApi = {
  register(payload: RegisterRequest) {
    return apiRequestOrThrow<RegisterResponse>('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  },

  login(payload: LoginRequest) {
    return apiRequestOrThrow<LoginResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  },

  logout() {
    return apiRequest<null>('/api/v1/auth/logout', {
      method: 'POST',
      auth: true,
    })
  },

  getMe(accessToken?: string) {
    return apiRequestOrThrow<UserProfileResponse>('/api/v1/users/me', {
      method: 'GET',
      auth: true,
      accessToken,
    })
  },
}
