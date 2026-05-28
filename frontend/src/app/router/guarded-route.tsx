import { Navigate, Outlet } from 'react-router-dom'
import { useAuthStore } from '@/features/auth/store/auth-store'

export function GuardedRoute() {
  const tokens = useAuthStore((state) => state.tokens)

  if (!tokens?.accessToken) {
    return <Navigate replace to="/auth" />
  }

  return <Outlet />
}
