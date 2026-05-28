import type { GroupListItem } from '@/shared/types/domain'
import { AvatarBadge } from '@/shared/components/avatar-badge'

interface GroupListProps {
  groups: GroupListItem[]
  onOpen: (conversationId: number) => void
}

export function GroupList({ groups, onOpen }: GroupListProps) {
  if (groups.length === 0) {
    return (
      <div className="panel-empty">
        <strong>还没有加入群聊</strong>
        <span>即使消息栏里隐藏了群聊，这里仍然会保留群入口。</span>
      </div>
    )
  }

  return (
    <div className="sidebar-list">
      {groups.map((group) => (
        <button
          key={group.id}
          type="button"
          className="sidebar-list__item"
          onClick={() => onOpen(group.id)}
        >
          <AvatarBadge name={group.name} tone="group" shape="round" />
          <div className="sidebar-list__meta">
            <div className="sidebar-list__title-row">
              <strong>{group.name}</strong>
              <span>{group.unreadCount > 0 ? `${group.unreadCount} 条未读` : '群聊'}</span>
            </div>
            <div className="sidebar-list__subtitle-row">
              <span className="truncate">
                {group.lastMessage?.content || '暂无消息'}
              </span>
            </div>
          </div>
        </button>
      ))}
    </div>
  )
}
