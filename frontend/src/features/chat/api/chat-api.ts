import { apiRequest, apiRequestOrThrow } from '@/shared/api/client'
import type {
  ConversationListResponse,
  GroupConversationResponse,
  GroupDetailEnvelopeDto,
  GroupMemberListResponse,
} from '@/features/chat/types/chat'

export const chatApi = {
  listConversations() {
    return apiRequestOrThrow<ConversationListResponse>('/api/v1/conversations', {
      method: 'GET',
      auth: true,
    })
  },

  listGroups() {
    return apiRequestOrThrow<ConversationListResponse>(
      '/api/v1/conversations/groups',
      {
        method: 'GET',
        auth: true,
      },
    )
  },

  hideConversation(conversationId: number) {
    return apiRequest(`/api/v1/conversations/${conversationId}/hide`, {
      method: 'POST',
      auth: true,
    })
  },

  createGroup(name: string, memberIds: number[]) {
    return apiRequestOrThrow<GroupConversationResponse>('/api/v1/conversations/groups', {
      method: 'POST',
      auth: true,
      body: JSON.stringify({
        name,
        member_ids: memberIds,
      }),
    })
  },

  getGroupDetail(conversationId: number) {
    return apiRequestOrThrow<GroupDetailEnvelopeDto>(`/api/v1/conversations/groups/${conversationId}`, {
      method: 'GET',
      auth: true,
    })
  },

  listGroupMembers(conversationId: number) {
    return apiRequestOrThrow<GroupMemberListResponse>(
      `/api/v1/conversations/groups/${conversationId}/members`,
      {
        method: 'GET',
        auth: true,
      },
    )
  },

  updateGroupName(conversationId: number, name: string) {
    return apiRequest<null>(`/api/v1/conversations/groups/${conversationId}/name`, {
      method: 'POST',
      auth: true,
      body: JSON.stringify({ name }),
    })
  },

  inviteGroupMembers(conversationId: number, memberIds: number[]) {
    return apiRequest<null>(`/api/v1/conversations/groups/${conversationId}/invite`, {
      method: 'POST',
      auth: true,
      body: JSON.stringify({
        member_ids: memberIds,
      }),
    })
  },

  removeGroupMember(conversationId: number, userId: number) {
    return apiRequest<null>(
      `/api/v1/conversations/groups/${conversationId}/members/${userId}/remove`,
      {
        method: 'POST',
        auth: true,
      },
    )
  },

  leaveGroup(conversationId: number) {
    return apiRequest<null>(`/api/v1/conversations/groups/${conversationId}/leave`, {
      method: 'POST',
      auth: true,
    })
  },

  dismissGroup(conversationId: number) {
    return apiRequest<null>(`/api/v1/conversations/groups/${conversationId}/dismiss`, {
      method: 'POST',
      auth: true,
    })
  },
}
