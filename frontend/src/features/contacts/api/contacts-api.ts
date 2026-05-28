import { apiRequestOrThrow } from '@/shared/api/client'
import type { FriendListResponse } from '@/features/contacts/types/contacts'

export const contactsApi = {
  listFriends() {
    return apiRequestOrThrow<FriendListResponse>('/api/v1/friends', {
      method: 'GET',
      auth: true,
    })
  },
}
