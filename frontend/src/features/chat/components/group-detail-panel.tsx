import { AvatarBadge } from '@/shared/components/avatar-badge'
import { GroupEditDialog } from '@/features/chat/components/group-edit-dialog'
import { GroupInviteDialog } from '@/features/chat/components/group-invite-dialog'
import { ConfirmDialog } from '@/shared/components/confirm-dialog'
import { SlidePanel } from '@/shared/components/slide-panel'
import type { GroupDetail, GroupMember } from '@/shared/types/domain'
import { useState } from 'react'

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
  friends: Array<{ userId: number; username: string; avatar: string; online: boolean; conversationId: number }>
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
      <div className="stack-sections">
        <section className="group-overview-card">
          <div className="profile-panel group-overview-card__profile">
            <AvatarBadge
              name={detail.name}
              avatar={detail.avatar}
              size="lg"
              shape="round"
              tone="group"
            />
            <div className="profile-panel__info">
              <strong>{detail.name}</strong>
              <span>这是你当前查看的群聊资料页</span>
            </div>
          </div>

          <div className="group-stat-grid">
            <div className="group-stat-card">
              <span>我的角色</span>
              <strong>{roleLabel(detail.myRole)}</strong>
            </div>
            <div className="group-stat-card">
              <span>群成员</span>
              <strong>{detail.memberCount}</strong>
            </div>
          </div>
        </section>

        {(canRename || canInvite) ? (
          <section className="group-section">
            <div className="group-section__header">
              <strong>群管理</strong>
              <span>常用操作</span>
            </div>
            <div className="group-action-grid">
              {canRename ? (
                <button
                  type="button"
                  className="group-action-card"
                  onClick={() => setRenameOpen(true)}
                >
                  <strong>修改群名</strong>
                  <span>更新会话和群资料中的名称</span>
                </button>
              ) : null}
              {canInvite ? (
                <button
                  type="button"
                  className="group-action-card"
                  onClick={() => setInviteOpen(true)}
                >
                  <strong>邀请成员</strong>
                  <span>从好友列表中选择成员加入群聊</span>
                </button>
              ) : null}
            </div>
          </section>
        ) : null}

        <section className="group-section">
          <div className="group-section__header">
            <strong>成员列表</strong>
            <span>{members.length} 人</span>
          </div>
          <div className="member-list member-list--group">
            {members.map((member) => (
              <div key={member.userId} className="member-list__item member-list__item--rich">
                <AvatarBadge
                  name={member.username}
                  avatar={member.avatar}
                  online={member.online}
                  shape="round"
                />
                <div className="member-list__meta">
                  <strong>{member.username}</strong>
                  <div className="member-list__meta-row">
                    <span className={`member-role-chip ${roleTone(member.role)}`}>{roleLabel(member.role)}</span>
                    <span>{member.online ? '在线' : '离线'}</span>
                  </div>
                </div>
                {canDismiss && member.role !== 1 ? (
                  <button
                    type="button"
                    className="text-link member-list__action"
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
          <section className="group-section group-section--danger">
            <div className="group-section__header">
              <strong>危险操作</strong>
              <span>请谨慎执行</span>
            </div>
            <div className="group-action-grid">
              {canLeave ? (
                <button
                  type="button"
                  className="group-action-card"
                  onClick={() => setLeaveConfirmOpen(true)}
                >
                  <strong>退出群聊</strong>
                  <span>退出后将不再接收这个群的新消息</span>
                </button>
              ) : null}
              {canDismiss ? (
                <button
                  type="button"
                  className="group-action-card group-action-card--danger"
                  onClick={() => setDismissConfirmOpen(true)}
                >
                  <strong>解散群聊</strong>
                  <span>解散后该群不可恢复，请谨慎操作</span>
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
          description={
            memberToRemove
              ? `确认将 ${memberToRemove.username} 移出当前群聊吗？`
              : ''
          }
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
