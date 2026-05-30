import { useMemo, useState, useTransition } from 'react'
import { useNavigate } from 'react-router-dom'
import { AccountHistory } from '@/features/auth/components/account-history'
import { useAuthActions } from '@/features/auth/hooks/use-auth-actions'
import { useAuthStore } from '@/features/auth/store/auth-store'
import { ChatBubbleIcon } from '@/shared/components/icons'
import { ApiError } from '@/shared/types/api'

type AuthMode = 'login' | 'register'

export function AuthPage() {
  const navigate = useNavigate()
  const rememberedAccounts = useAuthStore((state) => state.rememberedAccounts)
  const removeRememberedAccount = useAuthStore((state) => state.removeRememberedAccount)
  const lastUsedUsername = useMemo(() => rememberedAccounts[0]?.username ?? '', [rememberedAccounts])
  const [mode, setMode] = useState<AuthMode>('login')
  const [rememberMe, setRememberMe] = useState(Boolean(lastUsedUsername))
  const [loginUsername, setLoginUsername] = useState(lastUsedUsername)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [errorMessage, setErrorMessage] = useState('')
  const [isPending, startTransition] = useTransition()
  const actions = useAuthActions()
  const activeUsername = mode === 'login' ? loginUsername : username

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setErrorMessage('')

    startTransition(() => {
      void (async () => {
        try {
          if (mode === 'login') {
            if (!loginUsername.trim()) {
              setErrorMessage('请输入用户名')
              return
            }
            await actions.login(loginUsername.trim(), password)
          } else {
            if (!username.trim()) {
              setErrorMessage('请输入用户名')
              return
            }
            await actions.register(username.trim(), password)
          }

          if (!rememberMe && mode === 'login' && loginUsername.trim()) {
            removeRememberedAccount(loginUsername.trim())
          }

          navigate('/chat', { replace: true })
        } catch (error) {
          if (error instanceof ApiError) {
            setErrorMessage(error.message)
            return
          }
          setErrorMessage('操作未完成，请稍后重试')
        }
      })()
    })
  }

  return (
    <main className="auth-studio">
      <header className="auth-studio__topbar">
        <div className="auth-studio__brand">
          <span className="auth-studio__brand-icon">
            <ChatBubbleIcon />
          </span>
          <strong>欢迎回来</strong>
        </div>
        <button type="button" className="auth-studio__locale">
          简体中文
        </button>
      </header>

      <section className="auth-studio__content">
        <div className="auth-panel">
          <header className="auth-panel__header">
            <h1>{mode === 'login' ? '登录' : '注册新账号'}</h1>
            <p>{mode === 'login' ? '登录您的账号开始聊天' : '创建账号后即可开始聊天'}</p>
          </header>

          <form className="auth-panel__form" onSubmit={handleSubmit}>
            <label className="field field--studio">
              <span>用户名</span>
              <input
                autoComplete="username"
                value={activeUsername}
                onChange={(event) => {
                  if (mode === 'login') {
                    setLoginUsername(event.target.value)
                  } else {
                    setUsername(event.target.value)
                  }
                }}
                placeholder="请输入用户名"
              />
            </label>

            <label className="field field--studio">
              <span>密码</span>
              <input
                type="password"
                autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder="请输入密码"
              />
            </label>

            {mode === 'login' ? (
              <div className="auth-panel__meta">
                <label className="auth-panel__checkbox">
                  <input
                    type="checkbox"
                    checked={rememberMe}
                    onChange={(event) => setRememberMe(event.target.checked)}
                  />
                  <span>记住我</span>
                </label>
                <button type="button" className="auth-panel__link">
                  忘记密码?
                </button>
              </div>
            ) : null}

            {errorMessage ? <p className="auth-form__error">{errorMessage}</p> : null}

            <button className="primary-button auth-panel__submit" disabled={isPending} type="submit">
              {isPending ? '处理中...' : mode === 'login' ? '登录' : '注册新账号'}
            </button>
          </form>
        </div>

        <div className="auth-panel auth-panel--ghost">
          <div className="auth-panel__switch-label">或</div>
          <button
            type="button"
            className="auth-panel__switch"
            onClick={() => {
              setMode(mode === 'login' ? 'register' : 'login')
              setErrorMessage('')
            }}
          >
            {mode === 'login' ? '注册新账号' : '返回登录'}
          </button>
        </div>

        {rememberedAccounts.length > 0 ? (
          <div className="auth-panel auth-panel--history">
            <AccountHistory
              accounts={rememberedAccounts}
              onPick={(pickedUsername) => {
                setMode('login')
                setLoginUsername(pickedUsername)
              }}
              onRemove={removeRememberedAccount}
            />
          </div>
        ) : null}
      </section>
    </main>
  )
}
