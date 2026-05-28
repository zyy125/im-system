import { useRef } from 'react'
import { ConfirmDialog } from '@/shared/components/confirm-dialog'
import { SlidePanel } from '@/shared/components/slide-panel'
import { resolveAssetUrl } from '@/shared/utils/asset-url'
import { useState } from 'react'

interface AvatarActionsPanelProps {
  open: boolean
  currentAvatar?: string
  uploading?: boolean
  errorMessage?: string
  onClose: () => void
  onUpload: (file: File) => Promise<void> | void
  onClear: () => Promise<void> | void
}

export function AvatarActionsPanel({
  open,
  currentAvatar,
  uploading = false,
  errorMessage,
  onClose,
  onUpload,
  onClear,
}: AvatarActionsPanelProps) {
  const inputRef = useRef<HTMLInputElement | null>(null)
  const [confirmClearOpen, setConfirmClearOpen] = useState(false)
  const currentAvatarUrl = resolveAssetUrl(currentAvatar)

  return (
    <>
      <SlidePanel open={open} title="头像管理" subtitle="上传或清空当前头像" onClose={onClose}>
        <div className="stack-actions">
          {currentAvatarUrl ? (
            <img className="avatar-preview" src={currentAvatarUrl} alt="当前头像" />
          ) : (
            <div className="avatar-preview avatar-preview--empty">暂无头像</div>
          )}
          <input
            ref={inputRef}
            type="file"
            accept="image/png,image/jpeg,image/webp"
            hidden
            onChange={(event) => {
              const file = event.target.files?.[0]
              if (file) {
                void onUpload(file)
              }
              event.currentTarget.value = ''
            }}
          />
          <button
            type="button"
            className="primary-button"
            disabled={uploading}
            onClick={() => inputRef.current?.click()}
          >
            {uploading ? '上传中...' : '上传头像'}
          </button>
          <button type="button" className="secondary-button" onClick={() => setConfirmClearOpen(true)}>
            清空头像
          </button>
          {errorMessage ? <p className="form-error">{errorMessage}</p> : null}
        </div>
      </SlidePanel>

      <ConfirmDialog
        open={confirmClearOpen}
        title="清空头像"
        description="清空后会恢复默认头像展示。"
        confirmText="确认清空"
        tone="danger"
        onClose={() => setConfirmClearOpen(false)}
        onConfirm={async () => {
          await onClear()
          setConfirmClearOpen(false)
        }}
      />
    </>
  )
}
