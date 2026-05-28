import { useQueryClient } from '@tanstack/react-query'
import { usersApi } from '@/features/users/api/users-api'
import { useAuthStore } from '@/features/auth/store/auth-store'

export function useUserActions() {
  const queryClient = useQueryClient()
  const setCurrentUser = useAuthStore((state) => state.setCurrentUser)

  return {
    async refreshCurrentUser() {
      const me = await usersApi.getOnlineStatus()
      const current = useAuthStore.getState().currentUser
      if (!current) {
        return
      }

      setCurrentUser({
        ...current,
        userId: me.user_id,
        avatar: me.avatar,
        online: me.online,
      })
    },

    async uploadAvatar(file: File) {
      await usersApi.uploadAvatar(file)
      await queryClient.invalidateQueries({ queryKey: ['auth', 'me'] })
      const current = useAuthStore.getState().currentUser
      if (current) {
        const user = await usersApi.getUser(current.userId)
        setCurrentUser({
          userId: user.user_id,
          avatar: user.avatar,
          username: user.username,
          online: user.online,
        })
      }
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['chat', 'conversations'] }),
        queryClient.invalidateQueries({ queryKey: ['contacts', 'friends'] }),
        queryClient.invalidateQueries({ queryKey: ['friend-requests'] }),
      ])
    },

    async clearAvatar() {
      await usersApi.clearAvatar()
      const current = useAuthStore.getState().currentUser
      if (current) {
        const user = await usersApi.getUser(current.userId)
        setCurrentUser({
          userId: user.user_id,
          avatar: user.avatar,
          username: user.username,
          online: user.online,
        })
      }
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['chat', 'conversations'] }),
        queryClient.invalidateQueries({ queryKey: ['contacts', 'friends'] }),
        queryClient.invalidateQueries({ queryKey: ['friend-requests'] }),
      ])
    },
  }
}
