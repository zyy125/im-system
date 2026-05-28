import { apiRequest, apiRequestOrThrow } from '@/shared/api/client'
import type { UserProfileResponse } from '@/features/auth/types/auth'

export const usersApi = {
  getUser(userId: number) {
    return apiRequestOrThrow<UserProfileResponse>(`/api/v1/users/${userId}`, {
      method: 'GET',
      auth: true,
    })
  },

  getOnlineStatus() {
    return apiRequestOrThrow<{
      user_id: number
      avatar: string
      online: boolean
    }>('/api/v1/user/online', {
      method: 'GET',
      auth: true,
    })
  },

  uploadAvatar(file: File) {
    const formData = new FormData()
    formData.append('file', file)
    return apiRequestOrThrow<{ avatar: string }>('/api/v1/users/avatar', {
      method: 'POST',
      auth: true,
      body: formData,
    })
  },

  clearAvatar() {
    return apiRequest<null>('/api/v1/users/avatar', {
      method: 'DELETE',
      auth: true,
    })
  },
}
