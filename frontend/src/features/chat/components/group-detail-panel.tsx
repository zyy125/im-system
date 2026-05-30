import { useState } from 'react'
import { GroupEditDialog } from '@/features/chat/components/group-edit-dialog'
import { GroupInviteDialog } from '@/features/chat/components/group-invite-dialog'
import { AvatarBadge } from '@/shared/components/avatar-badge'
import { ConfirmDialog } from '@/shared/components/confirm-dialog'
import {
  AddUserIcon,
  AlertTriangleIcon,
  ArrowRightIcon,
  PencilIcon,
} from '@/shared/components/icons'
import { SlidePanel } from '@/shared/components/slide-panel'
import type { GroupDetail, GroupMember } from '@/shared/types/domain'

const roleLabel = (role: number) => {
  switch (role) {
    case 1:
      return '群主'
    case 2:
      return '管理员'
    default:
      return '成员'
  }
}

const roleTone = (role: number) => {
  switch (role) {
    case 1:
      return 'is-owner'
    case 2:
      return 'is-admin'
    default:
      return ''
  }
}

interface GroupDetailPanelProps {
  detail: GroupDetail | null
  members: GroupMember[]
  friends: Array<{
    userId: number
    username: string
    avatar: string
    online: boolean
    conversationId: number
  }>
  open: boolean
  onClose: () => void
  onRename: (name: string) => Promise<void> | void
  onInvite: (userIds: number[]) => Promise<void> | void
  onRemoveMember: (userId: number) => Promise<void> | void
  onLeave: () => Promise<void> | void
  onDismiss: () => Promise<void> | void
}

export function GroupDetailPanel({
  detail,
  members,
  friends,
  open,
  onClose,
  onRename,
  onInvite,
  onRemoveMember,
  onLeave,
  onDismiss,
}: GroupDetailPanelProps) {
  const [renameOpen, setRenameOpen] = useState(false)
  const [inviteOpen, setInviteOpen] = useState(false)
  const [leaveConfirmOpen, setLeaveConfirmOpen] = useState(false)
  const [dismissConfirmOpen, setDismissConfirmOpen] = useState(false)
  const [memberToRemove, setMemberToRemove] = useState<GroupMember | null>(null)
  const [renameSubmitting, setRenameSubmitting] = useState(false)
  const [renameError, setRenameError] = useState('')
  const [inviteSubmitting, setInviteSubmitting] = useState(false)
  const [inviteError, setInviteError] = useState('')
  const canRename = detail?.myRole === 1 || detail?.myRole === 2
  const canInvite = detail?.myRole === 1 || detail?.myRole === 2
  const canDismiss = detail?.myRole === 1
  const canLeave = detail?.myRole !== 1

  if (!detail) {
    return (
      <SlidePanel open={open} title="群详情" onClose={onClose}>
        <div className="panel-empty">
          <strong>暂无群详情</strong>
          <span>请稍后重试。</span>
        </div>
      </SlidePanel>
    )
  }

  return (
    <SlidePanel
      open={open}
      title={detail.name}
      subtitle={`成员 ${detail.memberCount} · ${roleLabel(detail.myRole)}`}
      onClose={onClose}
    >
      <div className="group-detail-panel">
        <section className="group-detail-card">
          <div className="group-detail-card__hero">
            <AvatarBadge
              name={detail.name}
              avatar={detail.avatar}
              size="lg"
              shape="round"
              tone="group"
            />
            <div className="group-detail-card__copy">
              <strong>{detail.name}</strong>
              <span className="group-detail-card__eyebrow">账号 ID：{detail.id}</span>
              <div className="group-detail-card__tags">
                <span className="group-detail-tag group-detail-tag--accent">
                  {roleLabel(detail.myRole)}
                </span>
                <span className="group-detail-tag">已实名认证</span>
              </div>
            </div>
          </div>

          <div className="group-detail-card__stats">
            <div className="group-detail-card__stat">
              <span>我的角色</span>
              <strong>{roleLabel(detail.myRole)}</strong>
            </div>
            <div className="group-detail-card__stat">
              <span>成员数量</span>
              <strong>{detail.memberCount}</strong>
            </div>
          </div>
        </section>

        {(canRename || canInvite) ? (
          <section className="group-detail-block">
            <div className="group-detail-block__header">
              <strong>常用操作</strong>
            </div>
            <div className="group-detail-list">
              {canRename ? (
                <button
                  type="button"
                  className="group-detail-list__item"
                  onClick={() => setRenameOpen(true)}
                >
                  <span className="group-detail-list__icon">
                    <PencilIcon />
                  </span>
                  <span className="group-detail-list__copy">
                    <strong>修改昵称</strong>
                    <span>编辑你在群内的昵称</span>
                  </span>
                  <ArrowRightIcon />
                </button>
              ) : null}

              {canInvite ? (
                <button
                  type="button"
                  className="group-detail-list__item"
                  onClick={() => setInviteOpen(true)}
                >
                  <span className="group-detail-list__icon">
                    <AddUserIcon />
                  </span>
                  <span className="group-detail-list__copy">
                    <strong>邀请成员</strong>
                    <span>从好友列表中选择成员加入</span>
                  </span>
                  <ArrowRightIcon />
                </button>
              ) : null}
            </div>
          </section>
        ) : null}

        <section className="group-detail-block">
          <div className="group-detail-block__header">
            <strong>成员列表</strong>
          </div>
          <div className="group-member-card">
            {members.map((member, index) => (
              <div
                key={member.userId}
                className={
                  index === members.length - 1
                    ? 'group-member-card__row'
                    : 'group-member-card__row group-member-card__row--bordered'
                }
              >
                <AvatarBadge
                  name={member.username}
                  avatar={member.avatar}
                  online={member.online}
                  shape="round"
                />
                <div className="group-member-card__meta">
                  <strong>{member.username}</strong>
                  <div className="group-member-card__subline">
                    <span className={`member-role-chip ${roleTone(member.role)}`}>
                      {roleLabel(member.role)}
                    </span>
                    <span>{member.online ? (member.role === 1 ? '你' : '在线') : '未实名'}</span>
                  </div>
                </div>
                {canDismiss && member.role !== 1 ? (
                  <button
                    type="button"
                    className="group-member-card__action"
                    onClick={() => setMemberToRemove(member)}
                  >
                    移除
                  </button>
                ) : null}
              </div>
            ))}
          </div>
        </section>

        {(canLeave || canDismiss) ? (
          <section className="group-detail-block">
            <div className="group-detail-block__header">
              <strong>危险操作</strong>
            </div>
            <div className="group-detail-list group-detail-list--danger">
              {canLeave ? (
                <button
                  type="button"
                  className="group-detail-list__item group-detail-list__item--neutral"
                  onClick={() => setLeaveConfirmOpen(true)}
                >
                  <span className="group-detail-list__icon group-detail-list__icon--neutral">
                    <ArrowRightIcon />
                  </span>
                  <span className="group-detail-list__copy">
                    <strong>退出群聊</strong>
                    <span>退出后将不再接收该群的新消息</span>
                  </span>
                </button>
              ) : null}

              {canDismiss ? (
                <button
                  type="button"
                  className="group-detail-list__item group-detail-list__item--danger"
                  onClick={() => setDismissConfirmOpen(true)}
                >
                  <span className="group-detail-list__icon group-detail-list__icon--danger">
                    <AlertTriangleIcon />
                  </span>
                  <span className="group-detail-list__copy">
                    <strong>解散群聊</strong>
                    <span>解散后成员和消息关系将终止</span>
                  </span>
                </button>
              ) : null}
            </div>
          </section>
        ) : null}

        <GroupEditDialog
          open={renameOpen}
          title="修改群名"
          subtitle="新的名称会同步展示到会话列表和群详情"
          placeholder="输入新的群名称"
          confirmText="保存"
          initialValue={detail.name}
          submitting={renameSubmitting}
          errorMessage={renameError}
          onClose={() => setRenameOpen(false)}
          onSubmit={async (name) => {
            try {
              setRenameError('')
              setRenameSubmitting(true)
              await onRename(name)
              setRenameOpen(false)
            } catch (error) {
              setRenameError(error instanceof Error ? error.message : '修改群名失败')
            } finally {
              setRenameSubmitting(false)
            }
          }}
        />

        <GroupInviteDialog
          open={inviteOpen}
          friends={friends}
          submitting={inviteSubmitting}
          errorMessage={inviteError}
          onClose={() => setInviteOpen(false)}
          onSubmit={async (userIds) => {
            try {
              setInviteError('')
              setInviteSubmitting(true)
              await onInvite(userIds)
              setInviteOpen(false)
            } catch (error) {
              setInviteError(error instanceof Error ? error.message : '邀请成员失败')
            } finally {
              setInviteSubmitting(false)
            }
          }}
        />

        <ConfirmDialog
          open={leaveConfirmOpen}
          title="退出群聊"
          description="退出后将不再接收该群的新消息。"
          confirmText="确认退出"
          onClose={() => setLeaveConfirmOpen(false)}
          onConfirm={async () => {
            await onLeave()
            setLeaveConfirmOpen(false)
          }}
        />

        <ConfirmDialog
          open={dismissConfirmOpen}
          title="解散群聊"
          description="解散后该群将不可恢复，请谨慎操作。"
          confirmText="确认解散"
          tone="danger"
          onClose={() => setDismissConfirmOpen(false)}
          onConfirm={async () => {
            await onDismiss()
            setDismissConfirmOpen(false)
          }}
        />

        <ConfirmDialog
          open={Boolean(memberToRemove)}
          title="移除成员"
          description={memberToRemove ? `确认将 ${memberToRemove.username} 移出当前群聊吗？` : ''}
          confirmText="确认移除"
          tone="danger"
          onClose={() => setMemberToRemove(null)}
          onConfirm={async () => {
            if (!memberToRemove) {
              return
            }
            await onRemoveMember(memberToRemove.userId)
            setMemberToRemove(null)
          }}
        />
      </div>
    </SlidePanel>
  )
}
