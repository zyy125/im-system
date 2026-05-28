import { AvatarBadge } from '@/shared/components/avatar-badge'
import { SlidePanel } from '@/shared/components/slide-panel'
import type { UserProfile } from '@/shared/types/domain'

interface UserProfilePanelProps {
  user: UserProfile | null
  open: boolean
  onClose: () => void
}

export function UserProfilePanel({ user, open, onClose }: UserProfilePanelProps) {
  return (
    <SlidePanel
      open={open}
      title="用户资料"
      subtitle={user?.online ? '在线' : '离线'}
      onClose={onClose}
    >
      {user ? (
        <div className="stack-sections">
          <div className="profile-panel profile-panel--center">
            <AvatarBadge
              name={user.username}
              avatar={user.avatar}
              online={user.online}
              size="lg"
              shape="round"
            />
            <div className="profile-panel__info">
              <strong>{user.username}</strong>
              <span>{user.online ? '当前在线' : '当前离线'}</span>
            </div>
          </div>
        </div>
      ) : (
        <div className="panel-empty">
          <strong>未找到资料</strong>
          <span>请稍后重试。</span>
        </div>
      )}
    </SlidePanel>
  )
}
