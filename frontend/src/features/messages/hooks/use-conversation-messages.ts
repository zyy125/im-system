import { useMemo } from 'react'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { messagesApi } from '@/features/messages/api/messages-api'
import type { LatestReadState, MessageItem } from '@/shared/types/domain'
import { dedupeMessages, mapServerMessage } from '@/features/messages/utils/message-cache'

export function useConversationMessages(conversationId: number | null) {
  const openQuery = useQuery({
    queryKey: ['messages', 'open', conversationId],
    queryFn: () => messagesApi.openConversation(conversationId!),
    enabled: Boolean(conversationId),
  })

  const historyQuery = useInfiniteQuery({
    queryKey: ['messages', 'history', conversationId],
    queryFn: ({ pageParam }) => messagesApi.getHistory(conversationId!, pageParam),
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? lastPage.next_before_seq : undefined,
    enabled: Boolean(conversationId),
  })

  const messages = useMemo(() => {
    if (!historyQuery.data || !conversationId) {
      return [] as MessageItem[]
    }

    const pages = [...historyQuery.data.pages].reverse()
    return dedupeMessages(
      pages.flatMap((page) =>
        page.messages
          .filter((message) => message.conversation_id === conversationId)
          .map(mapServerMessage),
      ),
    )
  }, [conversationId, historyQuery.data])

  const latestReadState = useMemo(() => {
    if (!conversationId) {
      return null as LatestReadState | null
    }

    if (openQuery.data?.conversation.id !== conversationId) {
      return null as LatestReadState | null
    }

    const payload = openQuery.data?.latest_read_state
    if (!payload) {
      return null as LatestReadState | null
    }

    return {
      latestSentSeq: payload.latest_sent_seq,
      readByUserIds: payload.read_by_user_ids,
      readCount: payload.read_count,
    }
  }, [conversationId, openQuery.data])

  return {
    openQuery,
    historyQuery,
    messages,
    latestReadState,
  }
}
