import { Dialog } from '@/shared/components/dialog'

interface ConfirmDialogProps {
  open: boolean
  title: string
  description: string
  confirmText?: string
  tone?: 'default' | 'danger'
  onClose: () => void
  onConfirm: () => Promise<void> | void
}

export function ConfirmDialog({
  open,
  title,
  description,
  confirmText = '确认',
  tone = 'default',
  onClose,
  onConfirm,
}: ConfirmDialogProps) {
  return (
    <Dialog
      open={open}
      title={title}
      subtitle={description}
      onClose={onClose}
      footer={
        <>
          <button type="button" className="secondary-button" onClick={onClose}>
            取消
          </button>
          <button
            type="button"
            className={tone === 'danger' ? 'danger-button' : 'primary-button'}
            onClick={() => void onConfirm()}
          >
            {confirmText}
          </button>
        </>
      }
    />
  )
}
