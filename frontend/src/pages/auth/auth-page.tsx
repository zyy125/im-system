import { useMemo, useState, useTransition } from 'react'
import { useNavigate } from 'react-router-dom'
import { AccountHistory } from '@/features/auth/components/account-history'
import { useAuthActions } from '@/features/auth/hooks/use-auth-actions'
import { useAuthStore } from '@/features/auth/store/auth-store'
import { ApiError } from '@/shared/types/api'
import { normalizePublicIdInput, parsePublicId } from '@/shared/utils/public-id'

type AuthMode = 'login' | 'register'

export function AuthPage() {
  const navigate = useNavigate()
  const rememberedAccounts = useAuthStore((state) => state.rememberedAccounts)
  const removeRememberedAccount = useAuthStore(
    (state) => state.removeRememberedAccount,
  )
  const lastUsedPublicId = useMemo(
    () => rememberedAccounts[0]?.publicId?.toString() ?? '',
    [rememberedAccounts],
  )
  const [mode, setMode] = useState<AuthMode>('login')
  const [publicId, setPublicId] = useState(lastUsedPublicId)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [errorMessage, setErrorMessage] = useState('')
  const [isPending, startTransition] = useTransition()
  const actions = useAuthActions()

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setErrorMessage('')

    startTransition(() => {
      void (async () => {
        try {
          if (mode === 'login') {
            const parsed = parsePublicId(publicId)
            if (!parsed) {
              setErrorMessage('请输入有效的账号号。')
              return
            }

            await actions.login(parsed, password)
          } else {
            if (!username.trim()) {
              setErrorMessage('请输入展示名。')
              return
            }
            await actions.register(username.trim(), password)
          }

          navigate('/chat', { replace: true })
        } catch (error) {
          if (error instanceof ApiError) {
            setErrorMessage(error.message)
            return
          }
          setErrorMessage('操作未完成，请稍后重试。')
        }
      })()
    })
  }

  return (
    <main className="auth-layout">
      <section className="auth-hero">
        <div className="auth-hero__eyebrow">即时通讯系统</div>
        <h1>为长期聊天使用而设计的稳定中文桌面体验。</h1>
        <p>
          账号号用于精确登录，展示名用于自然交流。界面重点不是第一眼的噱头，
          而是长时间使用时的清晰、耐看和稳定。
        </p>
        <div className="auth-hero__detail">
          <span>账号号登录</span>
          <span>注册后自动进入</span>
          <span>支持多账号记忆</span>
        </div>
      </section>

      <section className="auth-panel">
        <div className="auth-panel__frame">
          <div className="auth-panel__tabs">
            <button
              type="button"
              className={mode === 'login' ? 'is-active' : ''}
              onClick={() => {
                setMode('login')
                setErrorMessage('')
              }}
            >
              登录
            </button>
            <button
              type="button"
              className={mode === 'register' ? 'is-active' : ''}
              onClick={() => {
                setMode('register')
                setErrorMessage('')
              }}
            >
              注册
            </button>
          </div>

          <form className="auth-form" onSubmit={handleSubmit}>
            {mode === 'login' ? (
              <label className="field">
                <span>账号号</span>
                <input
                  inputMode="numeric"
                  value={publicId}
                  onChange={(event) =>
                    setPublicId(normalizePublicIdInput(event.target.value))
                  }
                  placeholder="输入你的账号号"
                />
              </label>
            ) : (
              <label className="field">
                <span>展示名</span>
                <input
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  placeholder="别人会看到的名字"
                />
              </label>
            )}

            <label className="field">
              <span>密码</span>
              <input
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder={mode === 'login' ? '输入密码' : '设置密码'}
              />
            </label>

            {errorMessage ? <p className="auth-form__error">{errorMessage}</p> : null}

            <button className="primary-button" disabled={isPending} type="submit">
              {isPending
                ? '处理中...'
                : mode === 'login'
                  ? '进入聊天'
                  : '创建并进入'}
            </button>
          </form>

          <AccountHistory
            accounts={rememberedAccounts}
            onPick={(pickedPublicId) => {
              setMode('login')
              setPublicId(String(pickedPublicId))
            }}
            onRemove={removeRememberedAccount}
          />
        </div>
      </section>
    </main>
  )
}
