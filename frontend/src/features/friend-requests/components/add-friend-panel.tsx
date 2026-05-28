import { useState } from 'react'
import { AddUserIcon } from '@/shared/components/icons'

interface AddFriendPanelProps {
  pending?: boolean
  onSubmit: (payload: { username: string; message: string }) => Promise<void> | void
}

export function AddFriendPanel({ pending = false, onSubmit }: AddFriendPanelProps) {
  const [username, setUsername] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const trimmedUsername = username.trim()
    if (!trimmedUsername) {
      setError('请输入目标用户名')
      return
    }

    setError('')
    await onSubmit({
      username: trimmedUsername,
      message: message.trim(),
    })
    setUsername('')
    setMessage('')
  }

  return (
    <section className="add-friend-panel">
      <div className="add-friend-panel__header">
        <div className="add-friend-panel__icon">
          <AddUserIcon />
        </div>
        <div>
          <strong>添加好友</strong>
        </div>
      </div>

      <form className="add-friend-form" onSubmit={handleSubmit}>
        <label className="field field--compact">
          <span>目标用户名</span>
          <input
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            placeholder="输入对方用户名"
          />
        </label>

        <label className="field field--compact">
          <span>附言</span>
          <input
            value={message}
            onChange={(event) => setMessage(event.target.value)}
            placeholder="可选，例如：你好，我是..."
          />
        </label>

        {error ? <p className="auth-form__error">{error}</p> : null}

        <button className="primary-button" type="submit" disabled={pending}>
          {pending ? '发送中...' : '发送申请'}
        </button>
      </form>
    </section>
  )
}
