import type { SVGProps } from 'react'

type IconProps = SVGProps<SVGSVGElement>

export function MessageIcon(props: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" {...props}>
      <path d="M7 18.5L3.5 20l1.5-3.5V6.75C5 5.23 6.23 4 7.75 4h8.5C17.77 4 19 5.23 19 6.75v6.5c0 1.52-1.23 2.75-2.75 2.75H10z" />
      <path d="M8.5 8.5h7" />
      <path d="M8.5 11.5H13" />
    </svg>
  )
}

export function ContactsIcon(props: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" {...props}>
      <path d="M15.5 19.5v-1.25c0-1.8-1.7-3.25-3.8-3.25h-2.4c-2.1 0-3.8 1.45-3.8 3.25v1.25" />
      <circle cx="10.5" cy="8.5" r="3" />
      <path d="M17.5 8.5h3" />
      <path d="M19 7v3" />
    </svg>
  )
}

export function LogoutIcon(props: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" {...props}>
      <path d="M10 6.5H7.75A1.75 1.75 0 0 0 6 8.25v7.5c0 .97.78 1.75 1.75 1.75H10" />
      <path d="M13.5 8l4 4-4 4" />
      <path d="M17 12H9" />
    </svg>
  )
}

export function ChatBubbleIcon(props: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" {...props}>
      <path d="M7.5 18.25L4 19.75l1.5-3.5V7.25A3.25 3.25 0 0 1 8.75 4h6.5a3.25 3.25 0 0 1 3.25 3.25v5.5A3.25 3.25 0 0 1 15.25 16H10z" />
      <path d="M8.75 8.25h6.5" />
      <path d="M8.75 11.25H13.5" />
    </svg>
  )
}

export function AddUserIcon(props: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" {...props}>
      <circle cx="9.5" cy="8.5" r="3" />
      <path d="M4.5 18c.28-2.1 2.15-3.75 4.4-3.75h1.2c2.26 0 4.13 1.65 4.4 3.75" />
      <path d="M18 8v5" />
      <path d="M15.5 10.5h5" />
    </svg>
  )
}
