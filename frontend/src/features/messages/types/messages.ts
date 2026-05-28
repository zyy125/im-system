export interface MessageItemDto {
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

export interface MessageHistoryResponse {
  messages: MessageItemDto[]
  has_more: boolean
  next_before_seq?: number
}

export interface MessageSyncResponse {
  messages: MessageItemDto[]
  has_more: boolean
  max_returned_seq: number
}

export interface OpenConversationResponse {
  conversation: {
    id: number
    type: 1 | 2
    name: string
    unread_count: number
    peer?: {
      user_id: number
      avatar: string
      username: string
      online: boolean
    }
    last_message?: MessageItemDto
  }
  latest_read_state?: {
    latest_sent_seq: number
    read_by_user_ids: number[]
    read_count: number
  }
}
