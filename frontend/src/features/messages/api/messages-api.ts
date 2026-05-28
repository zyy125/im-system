import { apiRequestOrThrow } from '@/shared/api/client'
import type {
  MessageHistoryResponse,
  MessageSyncResponse,
  OpenConversationResponse,
} from '@/features/messages/types/messages'

export const messagesApi = {
  openConversation(conversationId: number) {
    return apiRequestOrThrow<OpenConversationResponse>(
      `/api/v1/conversations/${conversationId}/open`,
      {
        method: 'POST',
        auth: true,
      },
    )
  },

  getHistory(conversationId: number, beforeSeq?: number) {
    const params = new URLSearchParams({
      conversation_id: String(conversationId),
      limit: '20',
    })

    if (beforeSeq) {
      params.set('before_seq', String(beforeSeq))
    }

    return apiRequestOrThrow<MessageHistoryResponse>(
      `/api/v1/messages/history?${params.toString()}`,
      {
        method: 'GET',
        auth: true,
      },
    )
  },

  syncMessages(conversationId: number, afterSeq = 0) {
    const params = new URLSearchParams({
      conversation_id: String(conversationId),
      after_seq: String(afterSeq),
      limit: '100',
    })

    return apiRequestOrThrow<MessageSyncResponse>(
      `/api/v1/messages/sync?${params.toString()}`,
      {
        method: 'GET',
        auth: true,
      },
    )
  },
}
