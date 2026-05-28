import { useMemo } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuthActions } from '@/features/auth/hooks/use-auth-actions'
import { useAuthStore } from '@/features/auth/store/auth-store'
import { ConversationList } from '@/features/chat/components/conversation-list'
import { useChatSidebarData } from '@/features/chat/hooks/use-chat-sidebar-data'
import { useChatShellStore } from '@/features/chat/store/chat-shell-store'
import { FriendList } from '@/features/contacts/components/friend-list'
import { GroupList } from '@/features/contacts/components/group-list'
import { useContactsData } from '@/features/contacts/hooks/use-contacts-data'
import { FriendRequestList } from '@/features/friend-requests/components/friend-request-list'
import { AddFriendPanel } from '@/features/friend-requests/components/add-friend-panel'
import { useFriendRequestsData } from '@/features/friend-requests/hooks/use-friend-requests-data'
import { ChatComposer } from '@/features/messages/components/chat-composer'
import { ChatMessageList } from '@/features/messages/components/chat-message-list'
import { useConversationMessages } from '@/features/messages/hooks/use-conversation-messages'
import { useChatRealtime } from '@/features/messages/hooks/use-chat-realtime'
import { useMessageComposerStore } from '@/features/messages/store/message-composer-store'
import type { MessageItem } from '@/shared/types/domain'
import { chatWsClient } from '@/shared/ws/client'
import {
  ChatBubbleIcon,
  ContactsIcon,
  LogoutIcon,
  MessageIcon,
} from '@/shared/components/icons'

export function ChatPage() {
  const queryClient = useQueryClient()
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
  const { conversationsQuery, groupsQuery } = useChatSidebarData()
  const { friendsQuery } = useContactsData()
  const { incomingQuery, outgoingQuery, acceptMutation, rejectMutation, sendMutation } =
    useFriendRequestsData()
  const drafts = useMessageComposerStore((state) => state.drafts)
  const setDraft = useMessageComposerStore((state) => state.setDraft)

  const conversations = useMemo(
    () => conversationsQuery.data ?? [],
    [conversationsQuery.data],
  )
  const friends = useMemo(() => friendsQuery.data ?? [], [friendsQuery.data])
  const groups = useMemo(() => groupsQuery.data ?? [], [groupsQuery.data])
  const incoming = useMemo(() => incomingQuery.data ?? [], [incomingQuery.data])
  const outgoing = useMemo(() => outgoingQuery.data ?? [], [outgoingQuery.data])
  const {
    messages: historyMessages,
    historyQuery,
    latestReadState,
    openQuery,
  } = useConversationMessages(activeConversationId)
  const { localMessages, setLocalMessages, receiptState } = useChatRealtime({
    activeConversationId,
    currentPublicId: currentUser?.publicId,
    conversations,
  })

  const activeConversation = useMemo(
    () => conversations.find((conversation) => conversation.id === activeConversationId) ?? null,
    [activeConversationId, conversations],
  )

  const displayedMessages = useMemo(() => {
    if (!activeConversationId) {
      return [] as MessageItem[]
    }
    const pending = localMessages[activeConversationId] ?? []
    return [...historyMessages, ...pending].sort((left, right) => left.seq - right.seq)
  }, [activeConversationId, historyMessages, localMessages])

  const latestOwnMessage = useMemo(() => {
    if (!currentUser) {
      return null
    }

    return [...displayedMessages]
      .reverse()
      .find((message) => message.fromPublicId === currentUser.publicId) ?? null
  }, [currentUser, displayedMessages])

  const openConversation = async (conversationId: number) => {
    setActiveConversationId(conversationId)
    setPrimaryView('messages')
    await queryClient.invalidateQueries({ queryKey: ['chat'] })
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
      fromPublicId: currentUser.publicId,
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

  return (
    <main className="chat-stage">
      <aside className="chat-stage__rail">
        <div className="chat-stage__brand">IM</div>
        <nav className="chat-stage__rail-nav">
          <button
            className={primaryView === 'messages' ? 'is-active' : ''}
            type="button"
            onClick={() => setPrimaryView('messages')}
          >
            <MessageIcon className="nav-icon" />
            消息
          </button>
          <button
            className={primaryView === 'contacts' ? 'is-active' : ''}
            type="button"
            onClick={() => setPrimaryView('contacts')}
          >
            <ContactsIcon className="nav-icon" />
            联系人
          </button>
        </nav>
        <button
          className="chat-stage__logout"
          type="button"
          onClick={() => void logout()}
        >
          <LogoutIcon className="nav-icon" />
          退出登录
        </button>
      </aside>

      <section className="chat-stage__list">
        <div className="chat-stage__panel-header">
          <span>{primaryView === 'messages' ? '消息栏' : '联系人'}</span>
          <strong>{currentUser?.username ?? '未知用户'}</strong>
          <small>
            #{currentUser?.publicId ?? '-'}
            {currentUser?.online ? ' · 在线' : ''}
          </small>
        </div>

        {primaryView === 'messages' ? (
          <ConversationList
            conversations={conversations}
            activeConversationId={activeConversationId}
            onOpen={(conversationId) => void openConversation(conversationId)}
          />
        ) : (
          <>
            <div className="subnav-tabs">
              <button
                type="button"
                className={contactsView === 'friends' ? 'is-active' : ''}
                onClick={() => setContactsView('friends')}
              >
                好友
              </button>
              <button
                type="button"
                className={contactsView === 'groups' ? 'is-active' : ''}
                onClick={() => setContactsView('groups')}
              >
                群聊
              </button>
            </div>

            <AddFriendPanel
              pending={sendMutation.isPending}
              onSubmit={async (payload) => {
                await sendMutation.mutateAsync(payload)
              }}
            />

            {contactsView === 'friends' ? (
              <FriendList
                friends={friends}
                onOpen={(conversationId) => void openConversation(conversationId)}
              />
            ) : (
              <GroupList
                groups={groups}
                onOpen={(conversationId) => void openConversation(conversationId)}
              />
            )}
          </>
        )}
      </section>

      <section className="chat-stage__conversation">
        <div className="chat-stage__panel-header">
          <span>{activeConversation ? '当前会话' : '聊天区'}</span>
          <strong>{activeConversation?.name ?? '请选择一个会话'}</strong>
          <small>
            {activeConversation?.peer
              ? `#${activeConversation.peer.publicId}`
              : latestReadState
                ? `你最新一条消息已有 ${latestReadState.readCount} 人已读`
                : '实时消息和已读恢复即将接入'}
          </small>
        </div>

        {activeConversation && currentUser ? (
          <>
            <ChatMessageList
              messages={displayedMessages}
              currentPublicId={currentUser.publicId}
              hasMore={Boolean(historyQuery.hasNextPage)}
              isLoadingInitial={historyQuery.isLoading || openQuery.isLoading}
              isLoadingMore={historyQuery.isFetchingNextPage}
              onLoadMore={() => {
                void historyQuery.fetchNextPage()
              }}
            />
            {latestOwnMessage ? (
              <div className="chat-read-state">
                <span>
                  最新一条自己发送的消息
                  {latestOwnMessage.seq
                    ? `（seq ${latestOwnMessage.seq}）`
                    : ''}
                </span>
                <strong>
                  已送达 {receiptState[`${latestOwnMessage.conversationId}:${latestOwnMessage.seq}`]?.deliveredByPublicIds.length ?? 0}
                  人，已读 {receiptState[`${latestOwnMessage.conversationId}:${latestOwnMessage.seq}`]?.readByPublicIds.length ?? latestReadState?.readCount ?? 0}
                  人
                </strong>
              </div>
            ) : null}
            <ChatComposer
              value={drafts[activeConversation.id] ?? ''}
              disabled={sendMessageMutation.isPending}
              onChange={(value) => setDraft(activeConversation.id, value)}
              onSend={handleSend}
            />
          </>
        ) : (
          <div className="chat-empty-state">
            <div className="chat-empty-state__icon">
              <ChatBubbleIcon />
            </div>
            <strong>请选择一个会话继续</strong>
            <p>从左侧消息栏或联系人中选择一个对象，开始新的聊天。</p>
          </div>
        )}

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
        />
      </section>
    </main>
  )
}
