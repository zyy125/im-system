import type { AuthTokens, CurrentUser } from '@/shared/types/domain'

export interface RegisterRequest {
  username: string
  password: string
}

export interface RegisterResponse {
  user_id: number
  username: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface UserProfileResponse {
  user_id: number
  avatar: string
  username: string
  online: boolean
}

export interface AuthSession {
  tokens: AuthTokens
  user: CurrentUser
}
