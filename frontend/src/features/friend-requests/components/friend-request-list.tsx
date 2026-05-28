import type { FriendRequestItem } from '@/shared/types/domain'
import { AvatarBadge } from '@/shared/components/avatar-badge'

interface FriendRequestListProps {
  incoming: FriendRequestItem[]
  outgoing: FriendRequestItem[]
  pendingActionId: number | null
  onAccept: (requestId: number) => void
  onReject: (requestId: number) => void
  showEmpty?: boolean
}

export function FriendRequestList({
  incoming,
  outgoing,
  pendingActionId,
  onAccept,
  onReject,
  showEmpty = false,
}: FriendRequestListProps) {
  if (incoming.length === 0 && outgoing.length === 0) {
    if (!showEmpty) {
      return null
    }

    return (
      <section className="request-stack">
        <div className="request-stack__header">
          <span>我的申请</span>
        </div>
        <div className="panel-empty panel-empty--compact">
          <strong>还没有申请记录</strong>
          <span>你发出的申请和收到的申请，都会整理在这里。</span>
        </div>
      </section>
    )
  }

  return (
    <section className="request-stack">
      <div className="request-stack__header">
        <span>我的申请</span>
      </div>

      {incoming.length > 0 ? (
        <div className="request-stack__section">
          <div className="request-stack__label">
            <span>收到的申请</span>
            <strong>{incoming.length}</strong>
          </div>

          {incoming.map((request) => (
            <div key={`incoming-${request.id}`} className="request-card">
              <div className="request-card__profile">
                <AvatarBadge
                  name={request.requester.username}
                  avatar={request.requester.avatar}
                  online={request.requester.online}
                  size="sm"
                  shape="round"
                />
                <div>
                  <strong>{request.requester.username}</strong>
                  <small>{request.message || '向你发送了好友申请'}</small>
                </div>
              </div>
              <div className="request-card__actions">
                <button
                  type="button"
                  onClick={() => onAccept(request.id)}
                  disabled={pendingActionId === request.id}
                >
                  同意
                </button>
                <button
                  type="button"
                  className="is-muted"
                  onClick={() => onReject(request.id)}
                  disabled={pendingActionId === request.id}
                >
                  拒绝
                </button>
              </div>
            </div>
          ))}
        </div>
      ) : null}

      {outgoing.length > 0 ? (
        <div className="request-stack__section">
          <div className="request-stack__label">
            <span>我发出的申请</span>
            <strong>{outgoing.length}</strong>
          </div>

          {outgoing.map((request) => (
            <div key={`outgoing-${request.id}`} className="request-card is-outgoing">
              <div className="request-card__profile">
                <AvatarBadge
                  name={request.receiver.username}
                  avatar={request.receiver.avatar}
                  online={request.receiver.online}
                  size="sm"
                  shape="round"
                />
                <div>
                  <strong>{request.receiver.username}</strong>
                  <small>{request.message || '等待对方处理'}</small>
                </div>
              </div>
              <span className="request-card__status">待处理</span>
            </div>
          ))}
        </div>
      ) : null}
    </section>
  )
}
