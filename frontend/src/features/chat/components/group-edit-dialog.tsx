import { useEffect, useState } from 'react'
import { Dialog } from '@/shared/components/dialog'

interface GroupEditDialogProps {
  open: boolean
  title: string
  subtitle: string
  placeholder: string
  confirmText: string
  initialValue?: string
  submitting?: boolean
  errorMessage?: string
  onClose: () => void
  onSubmit: (value: string) => Promise<void> | void
}

export function GroupEditDialog({
  open,
  title,
  subtitle,
  placeholder,
  confirmText,
  initialValue = '',
  submitting = false,
  errorMessage,
  onClose,
  onSubmit,
}: GroupEditDialogProps) {
  const [value, setValue] = useState(initialValue)

  useEffect(() => {
    if (open) {
      setValue(initialValue)
    }
  }, [initialValue, open])

  return (
    <Dialog
      open={open}
      title={title}
      subtitle={subtitle}
      onClose={onClose}
      footer={
        <>
          <button type="button" className="secondary-button" onClick={onClose}>
            取消
          </button>
          <button
            type="button"
            className="primary-button"
            disabled={!value.trim() || submitting}
            onClick={() => void onSubmit(value.trim())}
          >
            {submitting ? '提交中...' : confirmText}
          </button>
        </>
      }
    >
      <label className="field field--compact">
        <span>{title}</span>
        <input
          value={value}
          onChange={(event) => setValue(event.target.value)}
          placeholder={placeholder}
        />
      </label>
      {errorMessage ? <p className="form-error">{errorMessage}</p> : null}
    </Dialog>
  )
}
