import { useEffect } from 'react'
import { RouterProvider } from 'react-router-dom'
import { appRouter } from '@/app/router/app-router'
import { AppProviders } from '@/app/providers/app-providers'
import { useAuthActions } from '@/features/auth/hooks/use-auth-actions'
import { useAuthStore } from '@/features/auth/store/auth-store'
import { AppShell } from '@/shared/components/app-shell'
import { ApiError } from '@/shared/types/api'
import { chatWsClient } from '@/shared/ws/client'

export function AppBootstrap() {
  const bootstrapped = useAuthStore((state) => state.bootstrapped)
  const hydrate = useAuthStore((state) => state.hydrate)
  const logout = useAuthStore((state) => state.logout)
  const tokens = useAuthStore((state) => state.tokens)
  const { restoreSession } = useAuthActions()

  useEffect(() => {
    hydrate()
  }, [hydrate])

  useEffect(() => {
    if (!bootstrapped) {
      return
    }

    void (async () => {
      try {
        await restoreSession()
      } catch (error) {
        if (error instanceof ApiError && error.code.startsWith('auth.')) {
          logout()
        }
      }
    })()
  }, [bootstrapped, logout, restoreSession])

  useEffect(() => {
    if (!tokens?.accessToken) {
      chatWsClient.disconnect()
      return
    }

    chatWsClient.connect()
    return () => {
      chatWsClient.disconnect()
    }
  }, [tokens?.accessToken])

  if (!bootstrapped) {
    return <div className="app-loading">正在准备工作区...</div>
  }

  return (
    <AppProviders>
      <AppShell>
        <RouterProvider router={appRouter} />
      </AppShell>
    </AppProviders>
  )
}
