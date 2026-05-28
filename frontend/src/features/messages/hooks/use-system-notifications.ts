import { useMemo, useState } from 'react'
import type { MessageItem, SystemNotification } from '@/shared/types/domain'

export function useSystemNotifications(messagesByConversation: Record<number, MessageItem[]>) {
  const [readIds, setReadIds] = useState<Set<string>>(new Set())

  const notifications = useMemo(() => {
    const items = new Map<string, SystemNotification>()

    Object.values(messagesByConversation).forEach((messages) => {
      messages.forEach((message) => {
        if (message.type !== 2 && !message.event) {
          return
        }

        const id = `${message.conversationId}:${message.seq}:${message.msgId}`
        items.set(id, {
          id,
          conversationId: message.conversationId,
          title: '系统通知',
          content: message.content,
          sendTime: message.sendTime,
          event: message.event,
          read: readIds.has(id),
        })
      })
    })

    return [...items.values()].sort((left, right) => right.sendTime - left.sendTime)
  }, [messagesByConversation, readIds])

  const unreadCount = notifications.filter((item) => !item.read).length

  return {
    notifications,
    unreadCount,
    markAllRead() {
      setReadIds((current) => {
        const next = new Set(current)
        notifications.forEach((item) => {
          next.add(item.id)
        })
        return next
      })
    },
    markRead(id: string) {
      setReadIds((current) => new Set(current).add(id))
    },
  }
}
