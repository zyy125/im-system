interface ChatComposerProps {
  value: string
  disabled?: boolean
  onChange: (value: string) => void
  onSend: () => void
}

export function ChatComposer({
  value,
  disabled = false,
  onChange,
  onSend,
}: ChatComposerProps) {
  return (
    <div className="chat-composer">
      <div className="chat-composer__input">
        <textarea
          value={value}
          disabled={disabled}
          placeholder="输入消息..."
          onChange={(event) => onChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && !event.shiftKey) {
              event.preventDefault()
              onSend()
            }
          }}
        />
        <button type="button" disabled={disabled || !value.trim()} onClick={onSend}>
          发送
        </button>
      </div>
    </div>
  )
}
