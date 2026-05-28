export interface ConversationPeerDto {
  public_id: number
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
  from_public_id: number
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
