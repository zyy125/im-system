import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { messagesApi } from '@/features/messages/api/messages-api'
import { mergeMessagePages } from '@/features/messages/utils/message-cache'
import { upsertConversationByIncomingMessage } from '@/features/chat/utils/conversation-runtime'
import type { ConversationListItem, MessageItem, ReadReceiptState } from '@/shared/types/domain'
import { chatWsClient } from '@/shared/ws/client'
import type {
  WsMessageDeliveredPayload,
  WsMessagePayload,
  WsMessageReadPayload,
  WsPresenceChangedPayload,
  WsSyncRequiredPayload,
} from '@/shared/ws/protocol'

interface UseChatRealtimeParams {
  activeConversationId: number | null
  currentUserId: number | undefined
  conversations: ConversationListItem[]
}

export function useChatRealtime({
  activeConversationId,
  currentUserId,
  conversations,
}: UseChatRealtimeParams) {
  const queryClient = useQueryClient()
  const [localMessages, setLocalMessages] = useState<Record<number, MessageItem[]>>(
    {},
  )
  const [receiptState, setReceiptState] = useState<Record<string, ReadReceiptState>>({})

  useEffect(() => {
    const cleanupSent = chatWsClient.on('message.sent', (payload) => {
      const message = payload as WsMessagePayload
      setLocalMessages((current) => {
        const items = current[message.conversation_id] ?? []
        return {
          ...current,
          [message.conversation_id]: items
            .map((item) =>
              item.msgId === message.msg_id
                ? {
                    ...item,
                    id: message.id,
                    seq: message.seq,
                    sendTime: message.send_time,
                    status: 'sent' as const,
                    optimistic: false,
                  }
                : item,
            )
            .filter((item) => !(item.msgId === message.msg_id && item.id !== 0 && item.localId)),
        }
      })
      queryClient.invalidateQueries({ queryKey: ['chat', 'conversations'] }).catch(() => undefined)
      queryClient.invalidateQueries({
        queryKey: ['messages', 'history', message.conversation_id],
      }).catch(() => undefined)
    })

    const cleanupCreated = chatWsClient.on('message.created', (payload) => {
      const message = payload as WsMessagePayload
      queryClient.setQueryData(
        ['messages', 'history', message.conversation_id],
        (current: unknown) =>
          mergeMessagePages(current as never, [
            {
              id: message.id,
              msg_id: message.msg_id,
              conversation_id: message.conversation_id,
              seq: message.seq,
              type: message.type,
              event: message.event,
              from_user_id: message.from_user_id,
              send_time: message.send_time,
              content: message.content,
              extra: message.extra,
            },
          ]),
      )

      if (message.conversation_id === activeConversationId) {
        setLocalMessages((current) => {
          const items = current[message.conversation_id] ?? []
          const exists = items.some((item) => item.msgId === message.msg_id)
          if (exists) {
            return current
          }

          return {
            ...current,
            [message.conversation_id]: [
              ...items,
              {
                id: message.id,
                msgId: message.msg_id,
                conversationId: message.conversation_id,
                seq: message.seq,
                type: message.type,
                event: message.event,
                fromUserId: message.from_user_id,
                sendTime: message.send_time,
                content: message.content,
                extra: message.extra,
                status: 'sent' as const,
              },
            ],
          }
        })

        chatWsClient.sendDelivered({
          conversation_id: message.conversation_id,
          user_id: currentUserId ?? 0,
          delivered_seq: message.seq,
        })
        chatWsClient.sendRead({
          conversation_id: message.conversation_id,
          user_id: currentUserId ?? 0,
          read_seq: message.seq,
        })
      }

      queryClient.setQueryData(['chat', 'conversations'], (current: unknown) => {
        if (!Array.isArray(current)) {
          return upsertConversationByIncomingMessage(
            conversations,
            message,
            activeConversationId,
          )
        }

        return upsertConversationByIncomingMessage(
          current as ConversationListItem[],
          message,
          activeConversationId,
        )
      })
    })

    const cleanupDelivered = chatWsClient.on('message.delivered', (payload) => {
      const event = payload as WsMessageDeliveredPayload
      if (!activeConversationId || event.conversation_id !== activeConversationId) {
        return
      }

      setReceiptState((current) => {
        const key = `${event.conversation_id}:${event.delivered_seq}`
        const previous = current[key] ?? {
          deliveredByUserIds: [],
          readByUserIds: [],
        }
        if (previous.deliveredByUserIds.includes(event.user_id)) {
          return current
        }

        return {
          ...current,
          [key]: {
            ...previous,
            deliveredByUserIds: [...previous.deliveredByUserIds, event.user_id],
          },
        }
      })
    })

    const cleanupRead = chatWsClient.on('message.read', (payload) => {
      const event = payload as WsMessageReadPayload
      if (!activeConversationId || event.conversation_id !== activeConversationId) {
        return
      }

      setReceiptState((current) => {
        const key = `${event.conversation_id}:${event.read_seq}`
        const previous = current[key] ?? {
          deliveredByUserIds: [],
          readByUserIds: [],
        }
        if (previous.readByUserIds.includes(event.user_id)) {
          return current
        }

        return {
          ...current,
          [key]: {
            ...previous,
            readByUserIds: [...previous.readByUserIds, event.user_id],
          },
        }
      })
    })

    const cleanupPresence = chatWsClient.on('presence.changed', (payload) => {
      const event = payload as WsPresenceChangedPayload

      queryClient.setQueryData(['contacts', 'friends'], (current: unknown) => {
        if (!Array.isArray(current)) {
          return current
        }

        return current.map((friend) =>
          friend.userId === event.user_id
            ? { ...friend, online: event.online }
            : friend,
        )
      })

      queryClient.setQueryData(['chat', 'conversations'], (current: unknown) => {
        if (!Array.isArray(current)) {
          return current
        }

        return current.map((conversation) =>
          conversation.peer?.userId === event.user_id
            ? {
                ...conversation,
                peer: {
                  ...conversation.peer,
                  online: event.online,
                },
              }
            : conversation,
        )
      })
    })

    const cleanupSyncRequired = chatWsClient.on('sync.required', (payload) => {
      const event = payload as WsSyncRequiredPayload
      if (!event.conversation_id) {
        queryClient.invalidateQueries({ queryKey: ['chat', 'conversations'] }).catch(
          () => undefined,
        )
        return
      }

      void (async () => {
        const cached = queryClient.getQueryData([
          'messages',
          'history',
          event.conversation_id,
        ]) as
          | {
              pages?: Array<{ messages: Array<{ seq: number }> }>
            }
          | undefined

        const latestSeq =
          cached?.pages?.[cached.pages.length - 1]?.messages?.[
            cached.pages[cached.pages.length - 1].messages.length - 1
          ]?.seq ?? 0

        try {
          const synced = await messagesApi.syncMessages(event.conversation_id!, latestSeq)

          queryClient.setQueryData(
            ['messages', 'history', event.conversation_id],
            (current: unknown) =>
              mergeMessagePages(current as never, synced.messages),
          )
          await queryClient.invalidateQueries({
            queryKey: ['chat', 'conversations'],
          })
        } catch {
          await queryClient.invalidateQueries({
            queryKey: ['messages', 'history', event.conversation_id],
          })
        }
      })()
    })

    const cleanupError = chatWsClient.on('error', () => {
      setLocalMessages((current) => {
        if (!activeConversationId) {
          return current
        }

        const items = current[activeConversationId] ?? []
        return {
          ...current,
          [activeConversationId]: items.map((item) =>
            item.status === 'sending' ? { ...item, status: 'failed' as const } : item,
          ),
        }
      })
    })

    return () => {
      cleanupSent()
      cleanupCreated()
      cleanupDelivered()
      cleanupRead()
      cleanupPresence()
      cleanupSyncRequired()
      cleanupError()
    }
  }, [activeConversationId, conversations, currentUserId, queryClient])

  useEffect(() => {
    const cleanup = chatWsClient.onConnectionChange((connected) => {
      if (!connected || !activeConversationId) {
        return
      }

      void queryClient.invalidateQueries({
        queryKey: ['messages', 'history', activeConversationId],
      })
      void queryClient.invalidateQueries({
        queryKey: ['chat', 'conversations'],
      })
    })

    return cleanup
  }, [activeConversationId, queryClient])

  return {
    localMessages,
    setLocalMessages,
    receiptState,
  }
}
