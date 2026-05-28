export interface FriendItemDto {
  public_id: number
  username: string
  online: boolean
  conversation_id: number
}

export interface FriendListResponse {
  friends: FriendItemDto[]
}
