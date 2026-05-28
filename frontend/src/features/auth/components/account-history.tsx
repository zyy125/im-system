import type { RememberedAccount } from '@/shared/types/domain'

interface AccountHistoryProps {
  accounts: RememberedAccount[]
  onPick: (publicId: number) => void
  onRemove: (publicId: number) => void
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
          <div key={account.publicId} className="account-history__item">
            <button
              type="button"
              className="account-history__pick"
              onClick={() => onPick(account.publicId)}
            >
              <span className="account-history__public-id">
                {account.publicId}
              </span>
              <span className="account-history__username">{account.username}</span>
            </button>

            <button
              type="button"
              className="account-history__remove"
              onClick={() => onRemove(account.publicId)}
              aria-label={`删除 ${account.publicId}`}
            >
              删除
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
