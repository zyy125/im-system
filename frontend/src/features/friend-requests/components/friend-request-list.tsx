import type { FriendRequestItem } from '@/shared/types/domain'

interface FriendRequestListProps {
  incoming: FriendRequestItem[]
  outgoing: FriendRequestItem[]
  pendingActionId: number | null
  onAccept: (requestId: number) => void
  onReject: (requestId: number) => void
}

export function FriendRequestList({
  incoming,
  outgoing,
  pendingActionId,
  onAccept,
  onReject,
}: FriendRequestListProps) {
  if (incoming.length === 0 && outgoing.length === 0) {
    return null
  }

  return (
    <section className="request-stack">
      <div className="request-stack__header">
        <span>好友申请</span>
      </div>

      {incoming.map((request) => (
        <div key={`incoming-${request.id}`} className="request-card">
          <div>
            <strong>{request.requester.username}</strong>
            <p>#{request.requester.publicId}</p>
            <small>{request.message || '向你发送了好友申请'}</small>
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

      {outgoing.map((request) => (
        <div key={`outgoing-${request.id}`} className="request-card is-outgoing">
          <div>
            <strong>{request.receiver.username}</strong>
            <p>#{request.receiver.publicId}</p>
            <small>{request.message || '等待对方处理'}</small>
          </div>
          <span className="request-card__status">待处理</span>
        </div>
      ))}
    </section>
  )
}
