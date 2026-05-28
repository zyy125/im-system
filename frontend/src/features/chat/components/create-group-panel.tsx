import { useState } from 'react'
import { AvatarBadge } from '@/shared/components/avatar-badge'
import { SlidePanel } from '@/shared/components/slide-panel'
import type { FriendListItem } from '@/shared/types/domain'

interface CreateGroupPanelProps {
  friends: FriendListItem[]
  open: boolean
  onClose: () => void
  onCreate: (name: string, memberIds: number[]) => Promise<void> | void
}

export function CreateGroupPanel({ friends, open, onClose, onCreate }: CreateGroupPanelProps) {
  const [name, setName] = useState('')
  const [selected, setSelected] = useState<number[]>([])
  const [search, setSearch] = useState('')

  const filteredFriends = friends.filter((friend) =>
    friend.username.toLowerCase().includes(search.trim().toLowerCase()),
  )

  return (
    <SlidePanel open={open} title="创建群聊" subtitle="选择成员并填写群名称" onClose={onClose}>
      <div className="stack-sections">
        <label className="field field--compact">
          <span>群名称</span>
          <input value={name} onChange={(event) => setName(event.target.value)} placeholder="输入群名称" />
        </label>

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

        <button
          type="button"
          className="primary-button"
          onClick={() => void onCreate(name.trim(), selected)}
          disabled={!name.trim()}
        >
          创建群聊
        </button>
      </div>
    </SlidePanel>
  )
}
