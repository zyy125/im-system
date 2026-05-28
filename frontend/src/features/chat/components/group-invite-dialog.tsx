import { useEffect, useState } from 'react'
import { AvatarBadge } from '@/shared/components/avatar-badge'
import { Dialog } from '@/shared/components/dialog'
import type { FriendListItem } from '@/shared/types/domain'

interface GroupInviteDialogProps {
  open: boolean
  friends: FriendListItem[]
  submitting?: boolean
  errorMessage?: string
  onClose: () => void
  onSubmit: (userIds: number[]) => Promise<void> | void
}

export function GroupInviteDialog({
  open,
  friends,
  submitting = false,
  errorMessage,
  onClose,
  onSubmit,
}: GroupInviteDialogProps) {
  const [selected, setSelected] = useState<number[]>([])
  const [search, setSearch] = useState('')

  useEffect(() => {
    if (open) {
      setSelected([])
      setSearch('')
    }
  }, [open])

  const filteredFriends = friends.filter((friend) =>
    friend.username.toLowerCase().includes(search.trim().toLowerCase()),
  )

  return (
    <Dialog
      open={open}
      title="邀请成员"
      subtitle="从好友列表中选择要邀请入群的成员"
      onClose={onClose}
      footer={
        <>
          <button type="button" className="secondary-button" onClick={onClose}>
            取消
          </button>
          <button
            type="button"
            className="primary-button"
            disabled={selected.length === 0 || submitting}
            onClick={() => void onSubmit(selected)}
          >
            {submitting ? '邀请中...' : '邀请'}
          </button>
        </>
      }
    >
      <div className="stack-sections">
        <label className="field field--compact">
          <span>选择好友</span>
          <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索好友" />
        </label>

        <div className="select-friend-list">
          {filteredFriends.map((friend) => {
          const active = selected.includes(friend.userId)
          return (
            <button
              key={friend.userId}
              type="button"
              className={active ? 'select-friend-list__item is-active' : 'select-friend-list__item'}
              onClick={() =>
                setSelected((current) =>
                  current.includes(friend.userId)
                    ? current.filter((item) => item !== friend.userId)
                    : [...current, friend.userId],
                )
              }
            >
              <AvatarBadge
                name={friend.username}
                avatar={friend.avatar}
                online={friend.online}
                size="sm"
                shape="round"
              />
              <div className="select-friend-list__meta">
                <strong>{friend.username}</strong>
                <span>{friend.online ? '在线' : '离线'}</span>
              </div>
              <span className={active ? 'select-friend-list__check is-active' : 'select-friend-list__check'} />
            </button>
          )
        })}
      </div>
      </div>
      {errorMessage ? <p className="form-error">{errorMessage}</p> : null}
    </Dialog>
  )
}
