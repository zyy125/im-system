export interface RememberedAccount {
  username: string
  lastLoginAt: string
}

export interface AuthTokens {
  accessToken: string
  refreshToken: string
  expiresIn: number
}

export interface CurrentUser {
  userId: number
  username: string
  avatar: string
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
  fromUserId: number
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
  userId: number
  avatar: string
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
  userId: number
  avatar: string
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

export interface UserProfile {
  userId: number
  avatar: string
  username: string
  online: boolean
}

export interface FriendRequestUser {
  userId: number
  avatar: string
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
  readByUserIds: number[]
  readCount: number
}

export interface ReadReceiptState {
  deliveredByUserIds: number[]
  readByUserIds: number[]
}

export interface GroupDetail {
  id: number
  name: string
  avatar: string
  ownerId: number
  status: number
  myRole: number
  memberCount: number
}

export interface GroupMember {
  userId: number
  avatar: string
  username: string
  role: number
  online: boolean
}

export interface SystemNotification {
  id: string
  conversationId: number
  title: string
  content: string
  sendTime: number
  event: string
  read: boolean
}
