export interface RoomSocketHandlers {
  onMessage: (event: any) => void
  onOpen?: () => void
  onClose?: () => void
  onError?: (e: Event) => void
}

export interface ReplyPayload {
  replyToMessageId?: number
  replyToSenderName?: string
  replyToPreview?: string
}

export class RoomSocket {
  private ws: WebSocket | null = null
  private reconnectTimer: number | null = null
  private destroyed = false

  constructor(private roomId: string, private token: string, private handlers: RoomSocketHandlers) {}

  connect() {
    this.destroyed = false
    const wsBase = import.meta.env.VITE_WS_BASE || '/ws'
    const protocol = location.protocol === 'https:' ? 'wss' : 'ws'
    const path = `${wsBase}/rooms/${this.roomId}?token=${encodeURIComponent(this.token)}`
    const url = `${protocol}://${location.host}${path}`
    this.ws = new WebSocket(url)

    this.ws.onopen = () => {
      this.handlers.onOpen?.()
    }

    this.ws.onmessage = (evt) => {
      try {
        const data = JSON.parse(evt.data)
        this.handlers.onMessage(data)
      } catch {
        // ignore
      }
    }

    this.ws.onerror = (e) => {
      this.handlers.onError?.(e)
    }

    this.ws.onclose = () => {
      this.handlers.onClose?.()
      if (!this.destroyed) {
        this.reconnectTimer = window.setTimeout(() => this.connect(), 1500)
      }
    }
  }

  sendChat(content: string, reply?: ReplyPayload, clientMsgId?: string) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return
    this.ws.send(
      JSON.stringify({
        type: 'chat',
        content,
        clientMsgId,
        replyToMessageId: reply?.replyToMessageId,
        replyToSenderName: reply?.replyToSenderName,
        replyToPreview: reply?.replyToPreview
      })
    )
  }

  close() {
    this.destroyed = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.ws?.close()
  }
}
