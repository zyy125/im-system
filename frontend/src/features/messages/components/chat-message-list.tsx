import { useEffect, useRef } from 'react'
import type { MessageItem } from '@/shared/types/domain'

interface ChatMessageListProps {
  messages: MessageItem[]
  currentPublicId: number
  hasMore: boolean
  isLoadingInitial: boolean
  isLoadingMore: boolean
  onLoadMore: () => void
}

const formatMessageTime = (timestamp: number) =>
  new Intl.DateTimeFormat('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  }).format(timestamp)

export function ChatMessageList({
  messages,
  currentPublicId,
  hasMore,
  isLoadingInitial,
  isLoadingMore,
  onLoadMore,
}: ChatMessageListProps) {
  const scrollRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    const node = scrollRef.current
    if (!node) {
      return
    }

    node.scrollTop = node.scrollHeight
  }, [messages.length])

  if (isLoadingInitial) {
    return <div className="message-feed message-feed--empty">正在加载会话消息...</div>
  }

  return (
    <div
      ref={scrollRef}
      className="message-feed"
      onScroll={(event) => {
        const target = event.currentTarget
        if (target.scrollTop < 24 && hasMore && !isLoadingMore) {
          onLoadMore()
        }
      }}
    >
      <div className="message-feed__status">
        {isLoadingMore ? '正在加载更早消息...' : hasMore ? '上滑加载历史消息' : '没有更早消息了'}
      </div>

      {messages.map((message) => {
        const isOwn = message.fromPublicId === currentPublicId
        return (
          <div
            key={message.localId ?? `${message.conversationId}-${message.seq}-${message.msgId}`}
            className={isOwn ? 'chat-row is-own' : 'chat-row'}
          >
            <div className={isOwn ? 'chat-bubble is-own' : 'chat-bubble'}>
              <div className="chat-bubble__content">{message.content}</div>
              <div className="chat-bubble__meta">
                <span>{formatMessageTime(message.sendTime)}</span>
                {isOwn && message.status ? (
                  <span>
                    {message.status === 'sending'
                      ? '发送中'
                      : message.status === 'failed'
                        ? '发送失败'
                        : '已发送'}
                  </span>
                ) : null}
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}
