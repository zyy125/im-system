import type { PropsWithChildren } from 'react'

export function AppShell({ children }: PropsWithChildren) {
  return (
    <div className="app-shell">
      <div className="app-shell__backdrop" />
      <div className="app-shell__grain" />
      {children}
    </div>
  )
}
