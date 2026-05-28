import type { PropsWithChildren, ReactNode } from 'react'

interface DialogProps extends PropsWithChildren {
  open: boolean
  title: string
  subtitle?: string
  footer?: ReactNode
  onClose: () => void
}

export function Dialog({ open, title, subtitle, footer, children, onClose }: DialogProps) {
  if (!open) {
    return null
  }

  return (
    <div className="dialog-backdrop" onClick={onClose}>
      <div className="dialog-card" onClick={(event) => event.stopPropagation()}>
        <header className="dialog-card__header">
          <div>
            <strong>{title}</strong>
            {subtitle ? <span>{subtitle}</span> : null}
          </div>
          <button type="button" className="icon-button" onClick={onClose} aria-label="关闭">
            ×
          </button>
        </header>
        <div className="dialog-card__body">{children}</div>
        {footer ? <footer className="dialog-card__footer">{footer}</footer> : null}
      </div>
    </div>
  )
}
