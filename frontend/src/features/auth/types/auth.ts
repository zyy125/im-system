import type { AuthTokens, CurrentUser } from '@/shared/types/domain'

export interface RegisterRequest {
  username: string
  password: string
}

export interface RegisterResponse {
  public_id: number
}

export interface LoginRequest {
  public_id: number
  password: string
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface UserProfileResponse {
  public_id: number
  username: string
  online: boolean
}

export interface AuthSession {
  tokens: AuthTokens
  user: CurrentUser
}
