import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { friendRequestsApi } from '@/features/friend-requests/api/friend-requests-api'
import type { FriendRequestItem } from '@/shared/types/domain'

const mapRequestItem = (request: {
  id: number
  status: 1 | 2 | 3
  message: string
  requester: { user_id: number; avatar: string; username: string; online: boolean }
  receiver: { user_id: number; avatar: string; username: string; online: boolean }
}): FriendRequestItem => ({
  id: request.id,
  status: request.status,
  message: request.message,
  requester: {
    userId: request.requester.user_id,
    avatar: request.requester.avatar,
    username: request.requester.username,
    online: request.requester.online,
  },
  receiver: {
    userId: request.receiver.user_id,
    avatar: request.receiver.avatar,
    username: request.receiver.username,
    online: request.receiver.online,
  },
})

export function useFriendRequestsData() {
  const queryClient = useQueryClient()

  const incomingQuery = useQuery({
    queryKey: ['friend-requests', 'incoming'],
    queryFn: async () => {
      const payload = await friendRequestsApi.listIncoming()
      return payload.requests.map(mapRequestItem)
    },
  })

  const outgoingQuery = useQuery({
    queryKey: ['friend-requests', 'outgoing'],
    queryFn: async () => {
      const payload = await friendRequestsApi.listOutgoing()
      return payload.requests.map(mapRequestItem)
    },
  })

  const refreshAfterAction = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['friend-requests'] }),
      queryClient.invalidateQueries({ queryKey: ['chat', 'conversations'] }),
      queryClient.invalidateQueries({ queryKey: ['contacts', 'friends'] }),
    ])
  }

  const acceptMutation = useMutation({
    mutationFn: (requestId: number) => friendRequestsApi.accept(requestId),
    onSuccess: refreshAfterAction,
  })

  const rejectMutation = useMutation({
    mutationFn: (requestId: number) => friendRequestsApi.reject(requestId),
    onSuccess: refreshAfterAction,
  })

  const sendMutation = useMutation({
    mutationFn: (payload: { username: string; message: string }) =>
      friendRequestsApi.send(payload.username, payload.message),
    onSuccess: refreshAfterAction,
  })

  return {
    incomingQuery,
    outgoingQuery,
    acceptMutation,
    rejectMutation,
    sendMutation,
  }
}
