import { useEffect, useRef } from 'react'
import type { MessageItem } from '@/shared/types/domain'

interface ChatMessageListProps {
  messages: MessageItem[]
  currentUserId: number
  latestOwnMessageKey?: string | null
  latestOwnReadCount?: number
  hasMore: boolean
  isLoadingInitial: boolean
  isLoadingMore: boolean
  onLoadMore: () => void
  onRetryMessage?: (message: MessageItem) => void
}

const formatMessageTime = (timestamp: number) =>
  new Intl.DateTimeFormat('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  }).format(timestamp)

const formatMessageDate = (timestamp: number) =>
  new Intl.DateTimeFormat('zh-CN', {
    month: 'long',
    day: 'numeric',
  }).format(timestamp)

export function ChatMessageList({
  messages,
  currentUserId,
  latestOwnMessageKey = null,
  latestOwnReadCount = 0,
  hasMore,
  isLoadingInitial,
  isLoadingMore,
  onLoadMore,
  onRetryMessage,
}: ChatMessageListProps) {
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const shouldStickToBottomRef = useRef(true)

  useEffect(() => {
    const node = scrollRef.current
    if (!node) {
      return
    }

    if (shouldStickToBottomRef.current) {
      node.scrollTop = node.scrollHeight
    }
  }, [messages.length])

  if (isLoadingInitial) {
    return <div className="message-feed message-feed--empty">正在加载会话消息...</div>
  }

  let previousDay = ''

  return (
    <div
      ref={scrollRef}
      className="message-feed"
      onScroll={(event) => {
        const target = event.currentTarget
        const distanceFromBottom =
          target.scrollHeight - target.scrollTop - target.clientHeight
        shouldStickToBottomRef.current = distanceFromBottom < 80
        if (target.scrollTop < 24 && hasMore && !isLoadingMore) {
          onLoadMore()
        }
      }}
    >
      <div className="message-feed__status">
        {isLoadingMore ? '正在加载更早消息...' : hasMore ? '上滑加载历史消息' : '没有更早消息了'}
      </div>

      {messages
        .filter((message) => message.type !== 2 && !message.event)
        .map((message) => {
        const isOwn = message.fromUserId === currentUserId
        const messageKey =
          message.localId ?? `${message.conversationId}-${message.seq}-${message.msgId}`
        const isLatestOwnMessage = isOwn && latestOwnMessageKey === messageKey
        const statusText =
          isOwn && message.status
            ? message.status === 'sending'
              ? '发送中'
              : message.status === 'failed'
                ? '发送失败'
                : ''
            : ''
        const currentDay = formatMessageDate(message.sendTime)
        const showDateDivider = currentDay !== previousDay
        previousDay = currentDay
          return (
          <div key={messageKey}>
            {showDateDivider ? (
              <div className="message-feed__divider">
                <span>{currentDay}</span>
              </div>
            ) : null}

            <div className={isOwn ? 'chat-row is-own' : 'chat-row'}>
              <div className={isOwn ? 'chat-bubble is-own' : 'chat-bubble'}>
                <div className="chat-bubble__content">{message.content}</div>
                <div className="chat-bubble__meta">
                  <span>{formatMessageTime(message.sendTime)}</span>
                  {statusText ? (
                    <button
                      type="button"
                      className="chat-bubble__meta-button"
                      disabled={message.status !== 'failed'}
                      onClick={() => {
                        if (message.status === 'failed' && onRetryMessage) {
                          onRetryMessage(message)
                        }
                      }}
                    >
                      {statusText}
                    </button>
                  ) : null}
                  {isLatestOwnMessage && latestOwnReadCount > 0 ? (
                    <span className="chat-bubble__read-state">
                      {latestOwnReadCount > 1 ? `${latestOwnReadCount} 人已读` : '已读'}
                    </span>
                  ) : null}
                </div>
              </div>
            </div>
          </div>
          )
        })}
    </div>
  )
}
