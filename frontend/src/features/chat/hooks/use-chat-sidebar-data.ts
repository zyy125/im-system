import { useQuery } from '@tanstack/react-query'
import { chatApi } from '@/features/chat/api/chat-api'
import type {
  ConversationItemDto,
  GroupDetailDto,
  GroupMemberDto,
  MessagePreviewDto,
} from '@/features/chat/types/chat'
import type {
  ConversationListItem,
  ConversationPeer,
  GroupDetail,
  GroupMember,
  GroupListItem,
  MessagePreview,
} from '@/shared/types/domain'

export const mapMessagePreview = (
  payload?: MessagePreviewDto,
): MessagePreview | null => {
  if (!payload) {
    return null
  }

  return {
    id: payload.id,
    msgId: payload.msg_id,
    conversationId: payload.conversation_id,
    seq: payload.seq,
    type: payload.type,
    event: payload.event,
    fromUserId: payload.from_user_id,
    sendTime: payload.send_time,
    content: payload.content,
    extra: payload.extra,
  }
}

export const mapPeer = (payload?: {
  user_id: number
  avatar: string
  username: string
  online: boolean
}): ConversationPeer | null => {
  if (!payload) {
    return null
  }

  return {
    userId: payload.user_id,
    avatar: payload.avatar,
    username: payload.username,
    online: payload.online,
  }
}

export const mapConversationItem = (
  item: ConversationItemDto,
): ConversationListItem => ({
  id: item.id,
  type: item.type,
  name: item.name,
  unreadCount: item.unread_count,
  peer: mapPeer(item.peer),
  lastMessage: mapMessagePreview(item.last_message),
})

export const mapGroupDetail = (group: GroupDetailDto): GroupDetail => ({
  id: group.id,
  name: group.name,
  avatar: group.avatar,
  ownerId: group.owner_id,
  status: group.status,
  myRole: group.my_role,
  memberCount: group.member_count,
})

export const mapGroupMember = (member: GroupMemberDto): GroupMember => ({
  userId: member.user_id,
  avatar: member.avatar,
  username: member.username,
  role: member.role,
  online: member.online,
})

export const sortConversationItems = <T extends ConversationListItem>(items: T[]) =>
  [...items].sort((left, right) => {
    const leftTime = left.lastMessage?.sendTime ?? 0
    const rightTime = right.lastMessage?.sendTime ?? 0

    if (leftTime !== rightTime) {
      return rightTime - leftTime
    }

    return right.id - left.id
  })

export function useChatSidebarData() {
  const conversationsQuery = useQuery({
    queryKey: ['chat', 'conversations'],
    queryFn: async () => {
      const payload = await chatApi.listConversations()
      return sortConversationItems(payload.conversations.map(mapConversationItem))
    },
  })

  const groupsQuery = useQuery({
    queryKey: ['chat', 'groups'],
    queryFn: async () => {
      const payload = await chatApi.listGroups()
      return sortConversationItems(
        payload.conversations.map(mapConversationItem),
      ) as GroupListItem[]
    },
  })

  return {
    conversationsQuery,
    groupsQuery,
  }
}
