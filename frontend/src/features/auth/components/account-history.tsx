import type { RememberedAccount } from '@/shared/types/domain'

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
        <span>历史账号</span>
        <span>{accounts.length}</span>
      </div>

      <div className="account-history__list">
        {accounts.map((account) => (
          <div key={account.username} className="account-history__item">
            <button
              type="button"
              className="account-history__pick"
              onClick={() => onPick(account.username)}
            >
              <span className="account-history__public-id">
                @{account.username}
              </span>
              <span className="account-history__username">最近登录</span>
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
