import { apiRequest, apiRequestOrThrow } from '@/shared/api/client'
import type { FriendRequestListResponse } from '@/features/friend-requests/types/friend-requests'

export const friendRequestsApi = {
  send(targetPublicId: number, message: string) {
    return apiRequest<{ result: string }>(`/api/v1/friend-requests/${targetPublicId}`, {
      method: 'POST',
      auth: true,
      body: JSON.stringify(message ? { message } : {}),
    })
  },

  listIncoming() {
    return apiRequestOrThrow<FriendRequestListResponse>(
      '/api/v1/friend-requests/incoming',
      {
        method: 'GET',
        auth: true,
      },
    )
  },

  listOutgoing() {
    return apiRequestOrThrow<FriendRequestListResponse>(
      '/api/v1/friend-requests/outgoing',
      {
        method: 'GET',
        auth: true,
      },
    )
  },

  accept(requestId: number) {
    return apiRequest(`/api/v1/friend-requests/${requestId}/accept`, {
      method: 'POST',
      auth: true,
    })
  },

  reject(requestId: number) {
    return apiRequest(`/api/v1/friend-requests/${requestId}/reject`, {
      method: 'POST',
      auth: true,
    })
  },
}
