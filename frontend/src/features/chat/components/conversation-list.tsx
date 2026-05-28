import type { ConversationListItem } from '@/shared/types/domain'
import { AvatarBadge } from '@/shared/components/avatar-badge'

interface ConversationListProps {
  conversations: ConversationListItem[]
  activeConversationId: number | null
  onOpen: (conversationId: number) => void
}

const formatPreviewTime = (timestamp?: number) => {
  if (!timestamp) {
    return ''
  }

  return new Intl.DateTimeFormat('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  }).format(timestamp)
}

export function ConversationList({
  conversations,
  activeConversationId,
  onOpen,
}: ConversationListProps) {
  if (conversations.length === 0) {
    return (
      <div className="panel-empty">
        <strong>还没有活跃会话</strong>
        <span>可以先去联系人里打开好友或群聊开始聊天。</span>
      </div>
    )
  }

  return (
    <div className="sidebar-list">
      {conversations.map((conversation) => {
        const unreadCount =
          conversation.id === activeConversationId ? 0 : conversation.unreadCount

        return (
          <button
            key={conversation.id}
            type="button"
            className={
              conversation.id === activeConversationId
                ? 'sidebar-list__item is-active'
                : 'sidebar-list__item'
            }
            onClick={() => onOpen(conversation.id)}
          >
            <AvatarBadge
              name={conversation.peer?.username || conversation.name}
              avatar={conversation.peer?.avatar}
              online={conversation.peer?.online}
              tone={conversation.peer ? 'user' : 'group'}
              shape="round"
            />
            <div className="sidebar-list__meta">
              <div className="sidebar-list__title-row">
                <strong>{conversation.name}</strong>
                <span>{formatPreviewTime(conversation.lastMessage?.sendTime)}</span>
              </div>
              <div className="sidebar-list__subtitle-row">
                <span className="truncate">
                  {conversation.lastMessage?.content || '暂无消息'}
                </span>
              </div>
            </div>
            {unreadCount > 0 ? (
              <span className="sidebar-list__badge">{unreadCount}</span>
            ) : null}
          </button>
        )
      })}
    </div>
  )
}
