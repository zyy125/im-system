export type WsEventType =
  | 'message.send'
  | 'message.sent'
  | 'message.created'
  | 'message.delivered'
  | 'message.read'
  | 'presence.changed'
  | 'sync.required'
  | 'error'

export interface WsEnvelope<T = unknown> {
  type: WsEventType
  data: T
}

export interface WsMessageSendPayload {
  msg_id: string
  conversation_id: number
  content: string
}

export interface WsMessagePayload {
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

export interface WsMessageDeliveredPayload {
  conversation_id: number
  user_id: number
  delivered_seq: number
}

export interface WsMessageReadPayload {
  conversation_id: number
  user_id: number
  read_seq: number
}

export interface WsPresenceChangedPayload {
  user_id: number
  online: boolean
}

export interface WsSyncRequiredPayload {
  conversation_id?: number
  reason: string
}

export interface WsErrorPayload {
  code: string
  message: string
}
