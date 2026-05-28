export interface FriendItemDto {
  user_id: number
  avatar: string
  username: string
  online: boolean
  conversation_id: number
}

export interface FriendListResponse {
  friends: FriendItemDto[]
}
