export interface FriendRequestUserDto {
  public_id: number
  username: string
  online: boolean
}

export interface FriendRequestItemDto {
  id: number
  status: 1 | 2 | 3
  message: string
  requester: FriendRequestUserDto
  receiver: FriendRequestUserDto
}

export interface FriendRequestListResponse {
  requests: FriendRequestItemDto[]
}
