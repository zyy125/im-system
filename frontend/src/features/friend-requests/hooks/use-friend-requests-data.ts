import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { friendRequestsApi } from '@/features/friend-requests/api/friend-requests-api'
import type { FriendRequestItem } from '@/shared/types/domain'

const mapRequestItem = (request: {
  id: number
  status: 1 | 2 | 3
  message: string
  requester: { public_id: number; username: string; online: boolean }
  receiver: { public_id: number; username: string; online: boolean }
}): FriendRequestItem => ({
  id: request.id,
  status: request.status,
  message: request.message,
  requester: {
    publicId: request.requester.public_id,
    username: request.requester.username,
    online: request.requester.online,
  },
  receiver: {
    publicId: request.receiver.public_id,
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
    mutationFn: (payload: { publicId: number; message: string }) =>
      friendRequestsApi.send(payload.publicId, payload.message),
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
