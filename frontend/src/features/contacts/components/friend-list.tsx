import type { FriendListItem } from '@/shared/types/domain'
import { AvatarBadge } from '@/shared/components/avatar-badge'

interface FriendListProps {
  friends: FriendListItem[]
  onOpen: (conversationId: number) => void
}

export function FriendList({ friends, onOpen }: FriendListProps) {
  if (friends.length === 0) {
    return (
      <div className="panel-empty">
        <strong>还没有好友</strong>
        <span>好友申请通过后，会在这里显示对应的单聊联系人。</span>
      </div>
    )
  }

  return (
    <div className="sidebar-list">
      {friends.map((friend) => (
        <button
          key={friend.userId}
          type="button"
          className="sidebar-list__item"
          onClick={() => onOpen(friend.conversationId)}
        >
          <AvatarBadge
            name={friend.username}
            avatar={friend.avatar}
            online={friend.online}
            shape="round"
          />
          <div className="sidebar-list__meta">
            <div className="sidebar-list__title-row">
              <strong>{friend.username}</strong>
            </div>
            <div className="sidebar-list__subtitle-row">
              <span>{friend.online ? '在线' : '离线'}</span>
            </div>
          </div>
        </button>
      ))}
    </div>
  )
}
