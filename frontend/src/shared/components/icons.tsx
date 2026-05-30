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

export function SearchIcon(props: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" {...props}>
      <circle cx="11" cy="11" r="6.25" />
      <path d="M16 16l4 4" />
    </svg>
  )
}

export function BellIcon(props: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" {...props}>
      <path d="M12 4.75a4.25 4.25 0 0 0-4.25 4.25v2.08c0 .86-.2 1.7-.58 2.46L6.2 15.5h11.6l-.97-1.96a5.57 5.57 0 0 1-.58-2.46V9A4.25 4.25 0 0 0 12 4.75Z" />
      <path d="M9.75 18a2.25 2.25 0 0 0 4.5 0" />
    </svg>
  )
}

export function PencilIcon(props: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" {...props}>
      <path d="M4 20h4.25l9.4-9.4a1.8 1.8 0 0 0 0-2.55l-1.7-1.7a1.8 1.8 0 0 0-2.55 0L4 15.75V20Z" />
      <path d="m12.25 7.75 4 4" />
    </svg>
  )
}

export function UsersIcon(props: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" {...props}>
      <path d="M5 18.5c.24-2.1 2.06-3.75 4.25-3.75h1.5c2.2 0 4.01 1.65 4.25 3.75" />
      <circle cx="10" cy="8.5" r="3.25" />
      <path d="M16 15.5c.34-.54.54-1.18.54-1.87 0-1.94-1.56-3.51-3.49-3.53" />
      <path d="M17.5 18.5c-.12-1.06-.64-2.01-1.42-2.7" />
    </svg>
  )
}

export function SparklesIcon(props: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" {...props}>
      <path d="m12 4 1.35 3.65L17 9l-3.65 1.35L12 14l-1.35-3.65L7 9l3.65-1.35L12 4Z" />
      <path d="m18.5 14 .7 1.8L21 16.5l-1.8.7-.7 1.8-.7-1.8-1.8-.7 1.8-.7.7-1.8Z" />
      <path d="m6 15 .9 2.35L9.25 18l-2.35.65L6 21l-.9-2.35L2.75 18l2.35-.65L6 15Z" />
    </svg>
  )
}

export function ArrowRightIcon(props: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" {...props}>
      <path d="M5 12h14" />
      <path d="m13 6 6 6-6 6" />
    </svg>
  )
}

export function AlertTriangleIcon(props: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" {...props}>
      <path d="M10.45 4.87 3.9 16.2A2 2 0 0 0 5.63 19h12.74a2 2 0 0 0 1.73-2.8L13.55 4.87a2 2 0 0 0-3.1 0Z" />
      <path d="M12 9v4.25" />
      <circle cx="12" cy="16.75" r=".75" fill="currentColor" stroke="none" />
    </svg>
  )
}
