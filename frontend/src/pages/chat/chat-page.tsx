import { useDeferredValue, useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuthActions } from '@/features/auth/hooks/use-auth-actions'
import { useAuthStore } from '@/features/auth/store/auth-store'
import { chatApi } from '@/features/chat/api/chat-api'
import { CreateGroupPanel } from '@/features/chat/components/create-group-panel'
import { ConversationList } from '@/features/chat/components/conversation-list'
import { GroupDetailPanel } from '@/features/chat/components/group-detail-panel'
import { useChatSidebarData } from '@/features/chat/hooks/use-chat-sidebar-data'
import { mapGroupDetail, mapGroupMember } from '@/features/chat/hooks/use-chat-sidebar-data'
import { useChatShellStore } from '@/features/chat/store/chat-shell-store'
import { FriendList } from '@/features/contacts/components/friend-list'
import { GroupList } from '@/features/contacts/components/group-list'
import { useContactsData } from '@/features/contacts/hooks/use-contacts-data'
import { AddFriendPanel } from '@/features/friend-requests/components/add-friend-panel'
import { FriendRequestList } from '@/features/friend-requests/components/friend-request-list'
import { useFriendRequestsData } from '@/features/friend-requests/hooks/use-friend-requests-data'
import { messagesApi } from '@/features/messages/api/messages-api'
import { ChatComposer } from '@/features/messages/components/chat-composer'
import { ChatMessageList } from '@/features/messages/components/chat-message-list'
import { useChatRealtime } from '@/features/messages/hooks/use-chat-realtime'
import { useConversationMessages } from '@/features/messages/hooks/use-conversation-messages'
import { useMessageComposerStore } from '@/features/messages/store/message-composer-store'
import { SystemNotificationPanel } from '@/features/messages/components/system-notification-panel'
import { useSystemNotifications } from '@/features/messages/hooks/use-system-notifications'
import { AvatarActionsPanel } from '@/features/users/components/avatar-actions-panel'
import { UserProfilePanel } from '@/features/users/components/user-profile-panel'
import { useUserActions } from '@/features/users/hooks/use-user-actions'
import { AvatarBadge } from '@/shared/components/avatar-badge'
import { ConfirmDialog } from '@/shared/components/confirm-dialog'
import { useToast } from '@/shared/components/toast-provider'
import {
  BellIcon,
  ChatBubbleIcon,
  ContactsIcon,
  LogoutIcon,
  MessageIcon,
  SearchIcon,
} from '@/shared/components/icons'
import type { GroupDetail, GroupMember, MessageItem, UserProfile } from '@/shared/types/domain'
import { chatWsClient } from '@/shared/ws/client'
import { contactsApi } from '@/features/contacts/api/contacts-api'
import { usersApi } from '@/features/users/api/users-api'

export function ChatPage() {
  const queryClient = useQueryClient()
  const toast = useToast()
  const currentUser = useAuthStore((state) => state.currentUser)
  const { logout } = useAuthActions()
  const {
    primaryView,
    contactsView,
    activeConversationId,
    setPrimaryView,
    setContactsView,
    setActiveConversationId,
  } = useChatShellStore()
  const drafts = useMessageComposerStore((state) => state.drafts)
  const setDraft = useMessageComposerStore((state) => state.setDraft)
  const { conversationsQuery, groupsQuery } = useChatSidebarData()
  const { friendsQuery } = useContactsData()
  const { uploadAvatar, clearAvatar } = useUserActions()
  const { incomingQuery, outgoingQuery, acceptMutation, rejectMutation, sendMutation } =
    useFriendRequestsData()
  const [search, setSearch] = useState('')
  const [friendToolsView, setFriendToolsView] = useState<'add-friend' | 'requests' | null>(null)
  const [userProfile, setUserProfile] = useState<UserProfile | null>(null)
  const [userProfileOpen, setUserProfileOpen] = useState(false)
  const [avatarPanelOpen, setAvatarPanelOpen] = useState(false)
  const [avatarActionError, setAvatarActionError] = useState('')
  const [groupPanelOpen, setGroupPanelOpen] = useState(false)
  const [groupCreateOpen, setGroupCreateOpen] = useState(false)
  const [removeFriendConfirmOpen, setRemoveFriendConfirmOpen] = useState(false)
  const [notificationPanelOpen, setNotificationPanelOpen] = useState(false)
  const [groupDetail, setGroupDetail] = useState<GroupDetail | null>(null)
  const [groupMembers, setGroupMembers] = useState<GroupMember[]>([])
  const deferredSearch = useDeferredValue(search.trim().toLowerCase())
  const lastReadSyncKeyRef = useRef<string>('')
  const lastDeliveredSyncKeyRef = useRef<string>('')
  const openingConversationIdRef = useRef<number | null>(null)

  const conversations = useMemo(() => conversationsQuery.data ?? [], [conversationsQuery.data])
  const groups = useMemo(() => groupsQuery.data ?? [], [groupsQuery.data])
  const friends = useMemo(() => friendsQuery.data ?? [], [friendsQuery.data])
  const incoming = useMemo(() => incomingQuery.data ?? [], [incomingQuery.data])
  const outgoing = useMemo(() => outgoingQuery.data ?? [], [outgoingQuery.data])
  const requestCount = incoming.length + outgoing.length
  const friendUserIds = useMemo(() => new Set(friends.map((friend) => friend.userId)), [friends])
  const visibleConversations = useMemo(
    () =>
      conversations.filter((conversation) =>
        conversation.peer ? friendUserIds.has(conversation.peer.userId) : true,
      ),
    [conversations, friendUserIds],
  )

  const {
    messages: historyMessages,
    historyQuery,
    latestReadState,
    openQuery,
  } = useConversationMessages(activeConversationId)
  const { localMessages, setLocalMessages, receiptState } = useChatRealtime({
    activeConversationId,
    currentUserId: currentUser?.userId,
    conversations,
  })
  const systemMessagesByConversation = useMemo(() => {
    const result: Record<number, MessageItem[]> = {}

    historyMessages
      .filter((message) => message.type === 2 || Boolean(message.event))
      .forEach((message) => {
        result[message.conversationId] ??= []
        result[message.conversationId].push(message)
      })

    Object.entries(localMessages).forEach(([conversationId, messages]) => {
      messages
        .filter((message) => message.type === 2 || Boolean(message.event))
        .forEach((message) => {
          const key = Number(conversationId)
          result[key] ??= []
          result[key].push(message)
        })
    })

    return result
  }, [historyMessages, localMessages])

  const { notifications, unreadCount, markAllRead, markRead } =
    useSystemNotifications(systemMessagesByConversation)

  const activeConversation = useMemo(
    () =>
      visibleConversations.find((conversation) => conversation.id === activeConversationId) ?? null,
    [activeConversationId, visibleConversations],
  )

  const canSendInActiveConversation = useMemo(() => {
    if (!activeConversation) {
      return false
    }

    if (!activeConversation.peer) {
      return true
    }

    return friendUserIds.has(activeConversation.peer.userId)
  }, [activeConversation, friendUserIds])

  const displayedMessages = useMemo(() => {
    if (!activeConversationId) {
      return [] as MessageItem[]
    }

    const pending = (localMessages[activeConversationId] ?? []).filter(
      (message) =>
        message.conversationId === activeConversationId &&
        message.type !== 2 &&
        !message.event,
    )
    const history = historyMessages.filter(
      (message) =>
        message.conversationId === activeConversationId &&
        message.type !== 2 &&
        !message.event,
    )
    return [...history, ...pending].sort((left, right) => left.seq - right.seq)
  }, [activeConversationId, historyMessages, localMessages])

  const latestOwnMessage = useMemo(() => {
    if (!currentUser) {
      return null
    }

    return [...displayedMessages]
      .reverse()
      .find((message) => message.fromUserId === currentUser.userId) ?? null
  }, [currentUser, displayedMessages])

  const latestOwnMessageKey = useMemo(() => {
    if (!latestOwnMessage) {
      return null
    }
    return (
      latestOwnMessage.localId ??
      `${latestOwnMessage.conversationId}-${latestOwnMessage.seq}-${latestOwnMessage.msgId}`
    )
  }, [latestOwnMessage])

  const latestOwnReadCount = useMemo(() => {
    if (!latestOwnMessage) {
      return 0
    }

    const receiptKey = `${latestOwnMessage.conversationId}:${latestOwnMessage.seq}`
    const readCountFromRealtime = receiptState[receiptKey]?.readByUserIds.length ?? 0
    if (readCountFromRealtime > 0) {
      return readCountFromRealtime
    }

    if (latestReadState?.latestSentSeq === latestOwnMessage.seq) {
      return latestReadState.readCount
    }

    return 0
  }, [latestOwnMessage, latestReadState, receiptState])

  const filteredConversations = useMemo(() => {
    if (!deferredSearch) {
      return visibleConversations.map((conversation) =>
        conversation.id === activeConversationId
          ? { ...conversation, unreadCount: 0 }
          : conversation,
      )
    }

    return visibleConversations
      .map((conversation) =>
        conversation.id === activeConversationId
          ? { ...conversation, unreadCount: 0 }
          : conversation,
      )
      .filter((conversation) => {
      const haystack = [
        conversation.name,
        conversation.peer?.username,
        conversation.lastMessage?.content,
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()

      return haystack.includes(deferredSearch)
    })
  }, [activeConversationId, deferredSearch, visibleConversations])

  const filteredFriends = useMemo(() => {
    if (!deferredSearch) {
      return friends
    }

    return friends.filter((friend) => `${friend.username}`.toLowerCase().includes(deferredSearch))
  }, [friends, deferredSearch])

  const filteredGroups = useMemo(() => {
    if (!deferredSearch) {
      return groups
    }

    return groups.filter((group) =>
      `${group.name} ${group.id}`.toLowerCase().includes(deferredSearch),
    )
  }, [groups, deferredSearch])

  const openConversation = async (conversationId: number) => {
    setPrimaryView('messages')
    openingConversationIdRef.current = conversationId

    await messagesApi.openConversation(conversationId)

    queryClient.setQueryData(['chat', 'conversations'], (current: unknown) => {
      if (!Array.isArray(current)) {
        return current
      }
      return current.map((conversation) =>
        conversation.id === conversationId
          ? { ...conversation, unreadCount: 0 }
          : conversation,
      )
    })
    queryClient.setQueryData(['chat', 'groups'], (current: unknown) => {
      if (!Array.isArray(current)) {
        return current
      }
      return current.map((conversation) =>
        conversation.id === conversationId
          ? { ...conversation, unreadCount: 0 }
          : conversation,
      )
    })
    lastReadSyncKeyRef.current = ''
    setActiveConversationId(conversationId)
    await queryClient.invalidateQueries({ queryKey: ['chat', 'conversations'] })
    await queryClient.invalidateQueries({ queryKey: ['chat', 'groups'] })
    requestAnimationFrame(() => {
      const feed = document.querySelector('.message-feed')
      if (feed instanceof HTMLElement) {
        feed.scrollTop = feed.scrollHeight
      }
    })
    openingConversationIdRef.current = null
  }

  useEffect(() => {
    if (!activeConversationId) {
      return
    }

    if (openingConversationIdRef.current === activeConversationId) {
      return
    }

    const stillVisible = visibleConversations.some(
      (conversation) => conversation.id === activeConversationId,
    )
    if (!stillVisible) {
      setActiveConversationId(null)
    }
  }, [activeConversationId, setActiveConversationId, visibleConversations])

  const openUserProfile = async (userId: number) => {
    const user = await usersApi.getUser(userId)
    setUserProfile({
      userId: user.user_id,
      avatar: user.avatar,
      username: user.username,
      online: user.online,
    })
    setUserProfileOpen(true)
  }

  const openGroupDetail = async (conversationId: number) => {
    const [detailPayload, membersPayload] = await Promise.all([
      chatApi.getGroupDetail(conversationId),
      chatApi.listGroupMembers(conversationId),
    ])

    setGroupDetail(mapGroupDetail(detailPayload.group))
    setGroupMembers(membersPayload.members.map(mapGroupMember))
    setGroupPanelOpen(true)
  }

  const sendMessageMutation = useMutation({
    mutationFn: async (payload: {
      conversationId: number
      content: string
      msgId: string
    }) => {
      chatWsClient.sendMessage({
        msg_id: payload.msgId,
        conversation_id: payload.conversationId,
        content: payload.content,
      })
      return payload
    },
    onError: (_, payload) => {
      setLocalMessages((current) => {
        const items = current[payload.conversationId] ?? []
        return {
          ...current,
          [payload.conversationId]: items.map((item) =>
            item.msgId === payload.msgId && item.status === 'sending'
              ? { ...item, status: 'failed' as const }
              : item,
          ),
        }
      })
    },
  })

  const removeFriendMutation = useMutation({
    mutationFn: (userId: number) => contactsApi.removeFriend(userId),
    onSuccess: async () => {
      setActiveConversationId(null)
      toast.push('好友已删除', 'success')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['contacts', 'friends'] }),
        queryClient.invalidateQueries({ queryKey: ['chat', 'conversations'] }),
      ])
    },
    onError: (error) => {
      toast.push(error instanceof Error ? error.message : '删除好友失败', 'error')
    },
  })

  const hideConversationMutation = useMutation({
    mutationFn: (conversationId: number) => chatApi.hideConversation(conversationId),
    onSuccess: async () => {
      setActiveConversationId(null)
      toast.push('会话已隐藏', 'success')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['chat', 'conversations'] }),
        queryClient.invalidateQueries({ queryKey: ['contacts', 'friends'] }),
      ])
    },
    onError: (error) => {
      toast.push(error instanceof Error ? error.message : '隐藏会话失败', 'error')
    },
  })

  const retryMessage = (message: MessageItem) => {
    if (!message.content.trim()) {
      return
    }

    const retryId = `retry-${Date.now()}`
    setLocalMessages((current) => {
      const items = current[message.conversationId] ?? []
      return {
        ...current,
        [message.conversationId]: items
          .filter((item) => item.msgId !== message.msgId)
          .concat({
            ...message,
            id: 0,
            localId: retryId,
            msgId: retryId,
            seq: Number.MAX_SAFE_INTEGER - Date.now(),
            status: 'sending',
            optimistic: true,
            sendTime: Date.now(),
          }),
      }
    })
    sendMessageMutation.mutate({
      conversationId: message.conversationId,
      content: message.content,
      msgId: retryId,
    })
    toast.push('正在重发消息', 'info')
  }

  const handleSend = () => {
    if (!activeConversationId || !currentUser) {
      return
    }

    const draft = drafts[activeConversationId]?.trim()
    if (!draft) {
      return
    }

    const localId = `local-${Date.now()}`
    const optimisticMessage: MessageItem = {
      id: 0,
      localId,
      msgId: localId,
      conversationId: activeConversationId,
      seq: Number.MAX_SAFE_INTEGER - Date.now(),
      type: 1,
      event: '',
      fromUserId: currentUser.userId,
      sendTime: Date.now(),
      content: draft,
      extra: null,
      optimistic: true,
      status: 'sending',
    }

    setLocalMessages((current) => {
      const items = current[activeConversationId] ?? []
      return {
        ...current,
        [activeConversationId]: [...items, optimisticMessage],
      }
    })
    setDraft(activeConversationId, '')
    sendMessageMutation.mutate({
      conversationId: activeConversationId,
      content: draft,
      msgId: localId,
    })
  }

  useEffect(() => {
    if (!activeConversationId || displayedMessages.length === 0) {
      return
    }

    const latestMessage = displayedMessages[displayedMessages.length - 1]
    if (!latestMessage || latestMessage.fromUserId === currentUser?.userId) {
      return
    }

    const deliveredKey = `${activeConversationId}:${latestMessage.seq}:delivered`
    if (lastDeliveredSyncKeyRef.current !== deliveredKey) {
      lastDeliveredSyncKeyRef.current = deliveredKey
      chatWsClient.sendDelivered({
        conversation_id: activeConversationId,
        user_id: currentUser?.userId ?? 0,
        delivered_seq: latestMessage.seq,
      })
    }

    const readKey = `${activeConversationId}:${latestMessage.seq}:read`
    if (lastReadSyncKeyRef.current === readKey) {
      return
    }
    lastReadSyncKeyRef.current = readKey

    chatWsClient.sendRead({
      conversation_id: activeConversationId,
      user_id: currentUser?.userId ?? 0,
      read_seq: latestMessage.seq,
    })
  }, [activeConversationId, currentUser?.userId, displayedMessages])

  return (
    <main className="wechat-shell">
      <aside className="wechat-sidebar">
        <div className="wechat-sidebar__main">
          <header className="wechat-sidebar__header">
            <h1>{primaryView === 'messages' ? '消息' : '联系人'}</h1>
            <div className="wechat-sidebar__actions">
              <button
                type="button"
                className={primaryView === 'messages' ? 'text-tab is-active' : 'text-tab'}
                onClick={() => setPrimaryView('messages')}
                aria-label="消息"
              >
                <MessageIcon />
              </button>
              <button
                type="button"
                className={primaryView === 'contacts' ? 'text-tab is-active' : 'text-tab'}
                onClick={() => setPrimaryView('contacts')}
                aria-label="联系人"
              >
                <ContactsIcon />
              </button>
              <button
                type="button"
                className="text-tab notification-trigger"
              onClick={() => {
                  setNotificationPanelOpen(true)
                }}
                aria-label="通知中心"
              >
                <BellIcon />
                {unreadCount > 0 ? <span className="notification-dot">{unreadCount}</span> : null}
              </button>
            </div>
          </header>

          <div className="wechat-search">
            <SearchIcon />
            <input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder={primaryView === 'messages' ? '搜索' : '搜索联系人或群聊'}
            />
          </div>

          {primaryView === 'messages' ? (
            <ConversationList
              conversations={filteredConversations}
              activeConversationId={activeConversationId}
              onOpen={(conversationId) => void openConversation(conversationId)}
            />
          ) : (
            <div className="wechat-contacts-panel">
              <div className="wechat-inline-tabs">
                <button
                  type="button"
                  className={contactsView === 'friends' ? 'is-active' : ''}
                  onClick={() => {
                    setContactsView('friends')
                  }}
                >
                  好友
                </button>
                <button
                  type="button"
                  className={contactsView === 'groups' ? 'is-active' : ''}
                  onClick={() => {
                    setContactsView('groups')
                    setFriendToolsView(null)
                  }}
                >
                  群聊
                </button>
              </div>

              {contactsView === 'friends' ? (
                <>
                  <section className="contacts-tools">
                    <button
                      type="button"
                      className={
                        friendToolsView === 'add-friend'
                          ? 'contacts-tool-card is-active'
                          : 'contacts-tool-card'
                      }
                      onClick={() =>
                        setFriendToolsView((current) =>
                          current === 'add-friend' ? null : 'add-friend',
                        )
                      }
                    >
                      <strong>添加好友</strong>
                    </button>
                    <button
                      type="button"
                      className={
                        friendToolsView === 'requests'
                          ? 'contacts-tool-card is-active'
                          : 'contacts-tool-card'
                      }
                      onClick={() =>
                        setFriendToolsView((current) =>
                          current === 'requests' ? null : 'requests',
                        )
                      }
                    >
                      <strong>我的申请</strong>
                      {requestCount > 0 ? (
                        <span className="contacts-tool-card__badge">{requestCount}</span>
                      ) : null}
                    </button>
                  </section>

                  {friendToolsView === 'add-friend' ? (
                    <AddFriendPanel
                      pending={sendMutation.isPending}
                      onSubmit={async (payload) => {
                        await sendMutation.mutateAsync(payload)
                      }}
                    />
                  ) : null}

                  {friendToolsView === 'requests' ? (
                    <FriendRequestList
                      incoming={incoming}
                      outgoing={outgoing}
                      pendingActionId={
                        (acceptMutation.variables as number | undefined) ??
                        (rejectMutation.variables as number | undefined) ??
                        null
                      }
                      onAccept={(requestId) => void acceptMutation.mutateAsync(requestId)}
                      onReject={(requestId) => void rejectMutation.mutateAsync(requestId)}
                      showEmpty
                    />
                  ) : null}

                  <section className="contacts-list-section">
                    <div className="contacts-list-section__header">
                      <span>好友列表</span>
                      <strong>{filteredFriends.length}</strong>
                    </div>
                    <FriendList
                      friends={filteredFriends}
                      onOpen={(conversationId) => void openConversation(conversationId)}
                    />
                  </section>
                </>
              ) : (
                <>
                  <section className="contacts-tools contacts-tools--single">
                    <button
                      type="button"
                      className={
                        groupCreateOpen
                          ? 'contacts-tool-card contacts-tool-card--action is-active'
                          : 'contacts-tool-card contacts-tool-card--action'
                      }
                      onClick={() => setGroupCreateOpen(true)}
                    >
                      <strong>新建群聊</strong>
                    </button>
                  </section>

                  <section className="contacts-list-section">
                    <div className="contacts-list-section__header">
                      <span>群聊列表</span>
                      <strong>{filteredGroups.length}</strong>
                    </div>
                    <GroupList
                      groups={filteredGroups}
                      onOpen={(conversationId) => void openConversation(conversationId)}
                    />
                  </section>
                </>
              )}
            </div>
          )}
        </div>

        {currentUser ? (
          <div className="wechat-sidebar__account">
            <button
              type="button"
              className="wechat-sidebar__account-main"
              onClick={() => setAvatarPanelOpen(true)}
            >
              <AvatarBadge
                name={currentUser.username}
                avatar={currentUser.avatar}
                size="sm"
                shape="round"
              />
              <div className="wechat-sidebar__account-meta">
                <strong>{currentUser.username}</strong>
              </div>
            </button>
            <button
              type="button"
              className="wechat-sidebar__account-logout"
              onClick={() => void logout()}
              aria-label="退出登录"
            >
              <LogoutIcon />
            </button>
          </div>
        ) : (
          <button type="button" className="wechat-sidebar__account" onClick={() => void logout()}>
            <span>退出登录</span>
            <LogoutIcon />
          </button>
        )}
      </aside>

      <section className="wechat-room">
        {activeConversation && currentUser ? (
          <>
            <header className="wechat-room__header">
              <div className="wechat-room__identity">
                <button
                  type="button"
                  className="avatar-trigger"
                  onClick={() => {
                    if (activeConversation.peer) {
                      void openUserProfile(activeConversation.peer.userId)
                    } else {
                      void openGroupDetail(activeConversation.id)
                    }
                  }}
                >
                  <AvatarBadge
                    name={activeConversation.peer?.username || activeConversation.name}
                    avatar={activeConversation.peer?.avatar}
                    online={activeConversation.peer?.online}
                    size="lg"
                    shape="round"
                    tone={activeConversation.peer ? 'soft' : 'group'}
                  />
                </button>
                <div>
                  <strong>{activeConversation.name}</strong>
                  <span>
                    {activeConversation.peer
                      ? activeConversation.peer.online
                        ? '在线'
                        : '离线'
                      : '群聊'}
                  </span>
                </div>
              </div>
              <div className="wechat-room__toolbar">
                {activeConversation.peer ? (
                  <>
                    <button
                      type="button"
                      className="secondary-button"
                      onClick={() => setRemoveFriendConfirmOpen(true)}
                    >
                      删除好友
                    </button>
                  </>
                ) : (
                  <>
                    <button
                      type="button"
                      className="secondary-button"
                      onClick={() => void openGroupDetail(activeConversation.id)}
                    >
                      群详情
                    </button>
                  </>
                )}
                <button
                  type="button"
                  className="secondary-button"
                  onClick={() => void hideConversationMutation.mutateAsync(activeConversation.id)}
                >
                  隐藏会话
                </button>
              </div>
            </header>

            <div className="wechat-room__body">
              <ChatMessageList
                key={activeConversation.id}
                messages={displayedMessages}
                currentUserId={currentUser.userId}
                latestOwnMessageKey={latestOwnMessageKey}
                latestOwnReadCount={latestOwnReadCount}
                hasMore={Boolean(historyQuery.hasNextPage)}
                isLoadingInitial={historyQuery.isLoading || openQuery.isLoading}
                isLoadingMore={historyQuery.isFetchingNextPage}
                onLoadMore={() => {
                  void historyQuery.fetchNextPage()
                }}
                onRetryMessage={retryMessage}
              />
            </div>

            <footer className="wechat-room__footer">
              <ChatComposer
                value={drafts[activeConversation.id] ?? ''}
                disabled={sendMessageMutation.isPending || !canSendInActiveConversation}
                onChange={(value) => setDraft(activeConversation.id, value)}
                onSend={handleSend}
              />
            </footer>
          </>
        ) : (
          <div className="wechat-empty">
            <div>
              <div className="wechat-empty__icon">
                <ChatBubbleIcon />
              </div>
              <strong>选择一个会话开始聊天</strong>
              <p>左侧会显示你最近的会话、联系人和群聊入口。</p>
            </div>
          </div>
        )}

        <UserProfilePanel
          user={userProfile}
          open={userProfileOpen}
          onClose={() => setUserProfileOpen(false)}
        />

        <AvatarActionsPanel
          open={avatarPanelOpen}
          currentAvatar={currentUser?.avatar}
          onClose={() => setAvatarPanelOpen(false)}
          onUpload={async (file) => {
            try {
              setAvatarActionError('')
              await uploadAvatar(file)
              setAvatarPanelOpen(false)
              toast.push('头像已更新', 'success')
            } catch (error) {
              const message = error instanceof Error ? error.message : '头像上传失败'
              setAvatarActionError(message)
              toast.push(message, 'error')
            }
          }}
          onClear={async () => {
            try {
              setAvatarActionError('')
              await clearAvatar()
              setAvatarPanelOpen(false)
              toast.push('头像已清空', 'success')
            } catch (error) {
              const message = error instanceof Error ? error.message : '清空头像失败'
              setAvatarActionError(message)
              toast.push(message, 'error')
            }
          }}
          errorMessage={avatarActionError}
        />

        <CreateGroupPanel
          open={groupCreateOpen}
          friends={friends}
          onClose={() => setGroupCreateOpen(false)}
          onCreate={async (name, memberIds) => {
            const created = await chatApi.createGroup(name, memberIds)
            setGroupCreateOpen(false)
            toast.push('群聊创建成功', 'success')
            await Promise.all([
              queryClient.invalidateQueries({ queryKey: ['chat', 'conversations'] }),
              queryClient.invalidateQueries({ queryKey: ['chat', 'groups'] }),
            ])
            await openConversation(created.conversation.id)
          }}
        />

        <GroupDetailPanel
          detail={groupDetail}
          members={groupMembers}
          friends={friends}
          open={groupPanelOpen}
          onClose={() => setGroupPanelOpen(false)}
          onRename={async (name) => {
            if (!groupDetail) return
            await chatApi.updateGroupName(groupDetail.id, name)
            toast.push('群名称已更新', 'success')
            await Promise.all([
              queryClient.invalidateQueries({ queryKey: ['chat', 'conversations'] }),
              queryClient.invalidateQueries({ queryKey: ['chat', 'groups'] }),
            ])
            await openGroupDetail(groupDetail.id)
          }}
          onInvite={async (userIds) => {
            if (!groupDetail) return
            await chatApi.inviteGroupMembers(groupDetail.id, userIds)
            toast.push('已邀请成员', 'success')
            await openGroupDetail(groupDetail.id)
          }}
          onRemoveMember={async (userId) => {
            if (!groupDetail) return
            await chatApi.removeGroupMember(groupDetail.id, userId)
            toast.push('成员已移除', 'success')
            await openGroupDetail(groupDetail.id)
          }}
          onLeave={async () => {
            if (!groupDetail) return
            await chatApi.leaveGroup(groupDetail.id)
            toast.push('已退出群聊', 'success')
            setGroupPanelOpen(false)
            setActiveConversationId(null)
            await Promise.all([
              queryClient.invalidateQueries({ queryKey: ['chat', 'conversations'] }),
              queryClient.invalidateQueries({ queryKey: ['chat', 'groups'] }),
            ])
          }}
          onDismiss={async () => {
            if (!groupDetail) return
            await chatApi.dismissGroup(groupDetail.id)
            toast.push('群聊已解散', 'success')
            setGroupPanelOpen(false)
            setActiveConversationId(null)
            await Promise.all([
              queryClient.invalidateQueries({ queryKey: ['chat', 'conversations'] }),
              queryClient.invalidateQueries({ queryKey: ['chat', 'groups'] }),
            ])
          }}
        />

        <SystemNotificationPanel
          notifications={notifications}
          open={notificationPanelOpen}
          onClose={() => {
            markAllRead()
            setNotificationPanelOpen(false)
          }}
          onMarkRead={markRead}
          onOpenConversation={openConversation}
        />

        <ConfirmDialog
          open={removeFriendConfirmOpen}
          title="删除好友"
          description={
            activeConversation?.peer
              ? `确认删除好友 ${activeConversation.peer.username} 吗？`
              : ''
          }
          confirmText="确认删除"
          tone="danger"
          onClose={() => setRemoveFriendConfirmOpen(false)}
          onConfirm={async () => {
            if (!activeConversation?.peer) {
              return
            }
            await removeFriendMutation.mutateAsync(activeConversation.peer.userId)
            setRemoveFriendConfirmOpen(false)
          }}
        />
      </section>
    </main>
  )
}
