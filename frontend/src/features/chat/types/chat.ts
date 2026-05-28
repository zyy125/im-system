export interface ConversationPeerDto {
  user_id: number
  avatar: string
  username: string
  online: boolean
}

export interface MessagePreviewDto {
  id: number
  msg_id: string
  conversation_id: number
  seq: number
  type: number
  event: string
  from_user_id: number
  send_time: number
  content: string
  extra: unknown
}

export interface ConversationItemDto {
  id: number
  type: 1 | 2
  name: string
  unread_count: number
  peer?: ConversationPeerDto
  last_message?: MessagePreviewDto
}

export interface ConversationListResponse {
  conversations: ConversationItemDto[]
}

export interface GroupDetailDto {
  id: number
  name: string
  avatar: string
  owner_id: number
  status: number
  my_role: number
  member_count: number
}

export interface GroupDetailEnvelopeDto {
  group: GroupDetailDto
}

export interface GroupMemberDto {
  user_id: number
  avatar: string
  username: string
  role: number
  online: boolean
}

export interface GroupMemberListResponse {
  members: GroupMemberDto[]
}

export interface GroupConversationResponse {
  conversation: ConversationItemDto
}
