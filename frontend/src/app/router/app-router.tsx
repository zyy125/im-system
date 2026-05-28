import { Navigate, createBrowserRouter } from 'react-router-dom'
import { GuardedRoute } from '@/app/router/guarded-route'
import { AuthPage } from '@/pages/auth/auth-page'
import { ChatPage } from '@/pages/chat/chat-page'

export const appRouter = createBrowserRouter([
  {
    path: '/',
    element: <Navigate replace to="/chat" />,
  },
  {
    path: '/auth',
    element: <AuthPage />,
  },
  {
    element: <GuardedRoute />,
    children: [
      {
        path: '/chat',
        element: <ChatPage />,
      },
    ],
  },
])
