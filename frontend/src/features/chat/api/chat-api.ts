import { apiRequest, apiRequestOrThrow } from '@/shared/api/client'
import type { ConversationListResponse } from '@/features/chat/types/chat'

export const chatApi = {
  listConversations() {
    return apiRequestOrThrow<ConversationListResponse>('/api/v1/conversations', {
      method: 'GET',
      auth: true,
    })
  },

  listGroups() {
    return apiRequestOrThrow<ConversationListResponse>(
      '/api/v1/conversations/groups',
      {
        method: 'GET',
        auth: true,
      },
    )
  },

  hideConversation(conversationId: number) {
    return apiRequest(`/api/v1/conversations/${conversationId}/hide`, {
      method: 'POST',
      auth: true,
    })
  },
}
