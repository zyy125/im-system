import { env } from '@/shared/utils/env'
import { tokenStorage } from '@/shared/storage/tokens'
import type {
  WsEnvelope,
  WsErrorPayload,
  WsEventType,
  WsMessageDeliveredPayload,
  WsMessagePayload,
  WsMessageReadPayload,
  WsMessageSendPayload,
  WsPresenceChangedPayload,
  WsSyncRequiredPayload,
} from '@/shared/ws/protocol'

type MessageHandlerMap = {
  'message.sent': WsMessagePayload
  'message.created': WsMessagePayload
  'message.delivered': WsMessageDeliveredPayload
  'message.read': WsMessageReadPayload
  'presence.changed': WsPresenceChangedPayload
  'sync.required': WsSyncRequiredPayload
  error: WsErrorPayload
}

type Handler<T> = (payload: T) => void

class ChatWsClient {
  private socket: WebSocket | null = null
  private reconnectTimer: number | null = null
  private handlers: Record<string, Set<Handler<unknown>>> = {}
  private manuallyClosed = false
  private connectionListeners = new Set<(connected: boolean) => void>()

  connect() {
    const accessToken = tokenStorage.get()?.accessToken
    if (!accessToken || this.socket) {
      return
    }

    const url = new URL(`${env.wsBaseUrl}/api/v1/ws/`)
    url.searchParams.set('token', accessToken)

    this.manuallyClosed = false
    this.socket = new WebSocket(url)

    this.socket.addEventListener('message', (event) => {
      const envelope = JSON.parse(event.data) as WsEnvelope
      this.dispatch(envelope.type, envelope.data)
    })

    this.socket.addEventListener('close', () => {
      this.socket = null
      this.connectionListeners.forEach((listener) => listener(false))
      if (!this.manuallyClosed) {
        this.reconnectTimer = window.setTimeout(() => this.connect(), 1500)
      }
    })

    this.socket.addEventListener('open', () => {
      this.connectionListeners.forEach((listener) => listener(true))
    })
  }

  disconnect() {
    this.manuallyClosed = true
    if (this.reconnectTimer) {
      window.clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.socket?.close()
    this.socket = null
  }

  sendMessage(payload: WsMessageSendPayload) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      throw new Error('ws.disconnected')
    }

    this.socket.send(
      JSON.stringify({
        type: 'message.send',
        data: payload,
      }),
    )
  }

  sendDelivered(payload: WsMessageDeliveredPayload) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return
    }

    this.socket.send(
      JSON.stringify({
        type: 'message.delivered',
        data: {
          conversation_id: payload.conversation_id,
          delivered_seq: payload.delivered_seq,
        },
      }),
    )
  }

  sendRead(payload: WsMessageReadPayload) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return
    }

    this.socket.send(
      JSON.stringify({
        type: 'message.read',
        data: {
          conversation_id: payload.conversation_id,
          read_seq: payload.read_seq,
        },
      }),
    )
  }

  on<K extends keyof MessageHandlerMap>(
    type: K,
    handler: Handler<MessageHandlerMap[K]>,
  ) {
    const handlers = this.handlers[type] ?? new Set<Handler<unknown>>()
    handlers.add(handler as Handler<unknown>)
    this.handlers[type] = handlers

    return () => {
      handlers.delete(handler as Handler<unknown>)
    }
  }

  onConnectionChange(listener: (connected: boolean) => void) {
    this.connectionListeners.add(listener)
    return () => {
      this.connectionListeners.delete(listener)
    }
  }

  private dispatch(type: WsEventType, payload: unknown) {
    const handlers = this.handlers[type]
    if (!handlers) {
      return
    }

    handlers.forEach((handler) => {
      handler(payload)
    })
  }
}

export const chatWsClient = new ChatWsClient()
