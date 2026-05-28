import type { InfiniteData } from '@tanstack/react-query'
import type {
  MessageHistoryResponse,
  MessageItemDto,
} from '@/features/messages/types/messages'
import type { MessageItem } from '@/shared/types/domain'

export const mapServerMessage = (message: MessageItemDto): MessageItem => ({
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
  status: 'sent',
})

export const dedupeMessages = (messages: MessageItem[]) => {
  const seen = new Set<string>()
  return messages.filter((message) => {
    const key = `${message.conversationId}:${message.msgId}`
    if (seen.has(key)) {
      return false
    }
    seen.add(key)
    return true
  })
}

export const mergeMessagePages = (
  current: InfiniteData<MessageHistoryResponse, number | undefined> | undefined,
  incoming: MessageItemDto[],
): InfiniteData<MessageHistoryResponse, number | undefined> | undefined => {
  if (!current || current.pages.length === 0) {
    return current
  }

  const lastPageIndex = current.pages.length - 1
  const lastPage = current.pages[lastPageIndex]
  const existing = lastPage.messages

  const existingKeys = new Set(
    existing.map((message) => `${message.conversation_id}:${message.msg_id}`),
  )
  const additions = incoming.filter((message) => {
    const key = `${message.conversation_id}:${message.msg_id}`
    return !existingKeys.has(key)
  })

  if (additions.length === 0) {
    return current
  }

  const nextPages = current.pages.map((page, index) =>
    index === lastPageIndex
      ? {
          ...page,
          messages: [...page.messages, ...additions].sort((left, right) => left.seq - right.seq),
        }
      : page,
  )

  return {
    ...current,
    pages: nextPages,
  }
}
