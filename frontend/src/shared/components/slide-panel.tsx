import type { PropsWithChildren, ReactNode } from 'react'

interface SlidePanelProps extends PropsWithChildren {
  open: boolean
  title: string
  subtitle?: string
  actions?: ReactNode
  onClose: () => void
}

export function SlidePanel({
  open,
  title,
  subtitle,
  actions,
  children,
  onClose,
}: SlidePanelProps) {
  if (!open) {
    return null
  }

  return (
    <div className="slide-panel-backdrop" onClick={onClose}>
      <aside className="slide-panel" onClick={(event) => event.stopPropagation()}>
        <header className="slide-panel__header">
          <div>
            <strong>{title}</strong>
            {subtitle ? <span>{subtitle}</span> : null}
          </div>
          <button type="button" className="icon-button" onClick={onClose} aria-label="关闭">
            ×
          </button>
        </header>
        {actions ? <div className="slide-panel__actions">{actions}</div> : null}
        <div className="slide-panel__body">{children}</div>
      </aside>
    </div>
  )
}
