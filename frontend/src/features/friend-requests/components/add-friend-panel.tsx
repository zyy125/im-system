import { useState } from 'react'
import { AddUserIcon } from '@/shared/components/icons'

interface AddFriendPanelProps {
  pending?: boolean
  onSubmit: (payload: { publicId: number; message: string }) => Promise<void> | void
}

export function AddFriendPanel({ pending = false, onSubmit }: AddFriendPanelProps) {
  const [publicId, setPublicId] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const parsed = Number(publicId.trim())
    if (!Number.isSafeInteger(parsed) || parsed <= 0) {
      setError('请输入有效的账号号')
      return
    }

    setError('')
    await onSubmit({
      publicId: parsed,
      message: message.trim(),
    })
    setPublicId('')
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
          <p>通过账号号发送好友申请，支持附言。</p>
        </div>
      </div>

      <form className="add-friend-form" onSubmit={handleSubmit}>
        <label className="field field--compact">
          <span>账号号</span>
          <input
            inputMode="numeric"
            value={publicId}
            onChange={(event) => setPublicId(event.target.value.replace(/[^\d]/g, ''))}
            placeholder="输入对方 public_id"
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
