import { SlidePanel } from '@/shared/components/slide-panel'
import type { SystemNotification } from '@/shared/types/domain'

interface SystemNotificationPanelProps {
  notifications: SystemNotification[]
  open: boolean
  onClose: () => void
  onOpenConversation: (conversationId: number) => Promise<void> | void
  onMarkRead: (id: string) => void
}

const formatTime = (timestamp: number) =>
  new Intl.DateTimeFormat('zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(timestamp)

export function SystemNotificationPanel({
  notifications,
  open,
  onClose,
  onOpenConversation,
  onMarkRead,
}: SystemNotificationPanelProps) {
  return (
    <SlidePanel open={open} title="通知中心" subtitle="查看系统消息与未读提醒" onClose={onClose}>
      <div className="notification-list">
        {notifications.length === 0 ? (
          <div className="panel-empty">
            <strong>暂无通知</strong>
            <span>系统消息会集中显示在这里。</span>
          </div>
        ) : (
          notifications.map((item) => (
            <button
              key={item.id}
              type="button"
              className={item.read ? 'notification-card' : 'notification-card is-unread'}
              onClick={async () => {
                onMarkRead(item.id)
                await onOpenConversation(item.conversationId)
                onClose()
              }}
            >
              <div className="notification-card__head">
                <strong>{item.title}</strong>
                <span>{formatTime(item.sendTime)}</span>
              </div>
              <p>{item.content}</p>
            </button>
          ))
        )}
      </div>
    </SlidePanel>
  )
}
