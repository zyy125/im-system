import type { ConversationListItem } from '@/shared/types/domain'
import type { WsMessagePayload } from '@/shared/ws/protocol'
import { mapMessagePreview, sortConversationItems } from '@/features/chat/hooks/use-chat-sidebar-data'

export const upsertConversationByIncomingMessage = (
  conversations: ConversationListItem[],
  message: WsMessagePayload,
  activeConversationId: number | null,
): ConversationListItem[] => {
  const targetIndex = conversations.findIndex(
    (conversation) => conversation.id === message.conversation_id,
  )

  if (targetIndex === -1) {
    const placeholder: ConversationListItem = {
      id: message.conversation_id,
      type: 1,
      name: `会话 ${message.conversation_id}`,
      unreadCount: activeConversationId === message.conversation_id ? 0 : 1,
      peer: null,
      lastMessage: mapMessagePreview({
        id: message.id,
        msg_id: message.msg_id,
        conversation_id: message.conversation_id,
        seq: message.seq,
        type: message.type,
        event: message.event,
        from_public_id: message.from_public_id,
        send_time: message.send_time,
        content: message.content,
        extra: message.extra,
      }),
    }

    return sortConversationItems([placeholder, ...conversations])
  }

  const next = conversations.map((conversation) =>
    conversation.id === message.conversation_id
      ? {
          ...conversation,
          unreadCount:
            message.conversation_id === activeConversationId
              ? 0
              : conversation.unreadCount + 1,
          lastMessage: mapMessagePreview({
            id: message.id,
            msg_id: message.msg_id,
            conversation_id: message.conversation_id,
            seq: message.seq,
            type: message.type,
            event: message.event,
            from_public_id: message.from_public_id,
            send_time: message.send_time,
            content: message.content,
            extra: message.extra,
          }),
        }
      : conversation,
  )

  return sortConversationItems(next)
}
