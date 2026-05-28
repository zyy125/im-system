export interface RememberedAccount {
  publicId: number
  username: string
  lastLoginAt: string
}

export interface AuthTokens {
  accessToken: string
  refreshToken: string
  expiresIn: number
}

export interface CurrentUser {
  publicId: number
  username: string
  online: boolean
}

export type ConversationType = 1 | 2

export type FriendRequestStatus = 1 | 2 | 3

export interface MessagePreview {
  id: number
  msgId: string
  conversationId: number
  seq: number
  type: number
  event: string
  fromPublicId: number
  sendTime: number
  content: string
  extra: unknown
}

export type LocalMessageStatus = 'sending' | 'sent' | 'failed'

export interface MessageItem extends MessagePreview {
  localId?: string
  status?: LocalMessageStatus
  optimistic?: boolean
}

export interface ConversationPeer {
  publicId: number
  username: string
  online: boolean
}

export interface ConversationListItem {
  id: number
  type: ConversationType
  name: string
  unreadCount: number
  peer: ConversationPeer | null
  lastMessage: MessagePreview | null
}

export interface FriendListItem {
  publicId: number
  username: string
  online: boolean
  conversationId: number
}

export interface GroupListItem {
  id: number
  type: ConversationType
  name: string
  unreadCount: number
  peer: ConversationPeer | null
  lastMessage: MessagePreview | null
}

export interface FriendRequestUser {
  publicId: number
  username: string
  online: boolean
}

export interface FriendRequestItem {
  id: number
  status: FriendRequestStatus
  message: string
  requester: FriendRequestUser
  receiver: FriendRequestUser
}

export interface LatestReadState {
  latestSentSeq: number
  readByPublicIds: number[]
  readCount: number
}

export interface ReadReceiptState {
  deliveredByPublicIds: number[]
  readByPublicIds: number[]
}
