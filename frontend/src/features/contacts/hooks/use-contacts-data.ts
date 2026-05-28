import { useQuery } from '@tanstack/react-query'
import { contactsApi } from '@/features/contacts/api/contacts-api'
import type { FriendListItem } from '@/shared/types/domain'

export function useContactsData() {
  const friendsQuery = useQuery({
    queryKey: ['contacts', 'friends'],
    queryFn: async () => {
      const payload = await contactsApi.listFriends()
      return payload.friends
        .map<FriendListItem>((friend) => ({
          userId: friend.user_id,
          avatar: friend.avatar,
          username: friend.username,
          online: friend.online,
          conversationId: friend.conversation_id,
        }))
        .sort((left, right) => Number(right.online) - Number(left.online) || left.userId - right.userId)
    },
  })

  return {
    friendsQuery,
  }
}
