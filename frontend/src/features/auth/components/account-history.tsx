import type { RememberedAccount } from '@/shared/types/domain'

const formatLastLogin = (value: string) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '最近登录'
  }

  const diff = Date.now() - date.getTime()
  const day = 24 * 60 * 60 * 1000
  if (diff < day) {
    return '刚刚登录'
  }

  const days = Math.floor(diff / day)
  return `${days}天前登录`
}

interface AccountHistoryProps {
  accounts: RememberedAccount[]
  onPick: (username: string) => void
  onRemove: (username: string) => void
}

export function AccountHistory({
  accounts,
  onPick,
  onRemove,
}: AccountHistoryProps) {
  if (accounts.length === 0) {
    return null
  }

  return (
    <div className="account-history">
      <div className="account-history__header">
        <span>最近登录</span>
        <span>{accounts.length} 个账号</span>
      </div>

      <div className="account-history__list">
        {accounts.map((account) => (
          <div key={account.username} className="account-history__item">
            <button
              type="button"
              className="account-history__pick"
              onClick={() => onPick(account.username)}
            >
              <span className="account-history__public-id">{account.username}</span>
              <span className="account-history__username">
                {formatLastLogin(account.lastLoginAt)}
              </span>
            </button>

            <button
              type="button"
              className="account-history__remove"
              onClick={() => onRemove(account.username)}
              aria-label={`删除 ${account.username}`}
            >
              删除
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
