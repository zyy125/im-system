import { resolveAssetUrl } from '@/shared/utils/asset-url'

interface AvatarBadgeProps {
  name: string
  avatar?: string
  size?: 'sm' | 'md' | 'lg'
  tone?: 'user' | 'group' | 'soft'
  shape?: 'round' | 'squircle'
  online?: boolean
}

export function AvatarBadge({
  name,
  avatar,
  size = 'md',
  tone = 'user',
  shape = 'round',
  online,
}: AvatarBadgeProps) {
  const label = name.trim().slice(0, 1).toUpperCase() || '?'
  const avatarUrl = resolveAssetUrl(avatar)

  return (
    <div
      className={[
        'avatar-badge',
        `avatar-badge--${size}`,
        `avatar-badge--${tone}`,
        `avatar-badge--${shape}`,
      ].join(' ')}
    >
      {avatarUrl ? <img src={avatarUrl} alt={name} /> : <span>{label}</span>}
      {typeof online === 'boolean' ? (
        <span className={online ? 'presence-dot is-online' : 'presence-dot'} />
      ) : null}
    </div>
  )
}
