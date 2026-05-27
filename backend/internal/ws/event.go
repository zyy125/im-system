package ws

const (
	EventTypeMessageSend      = "message.send"
	EventTypeMessageSent      = "message.sent"
	EventTypeMessageDelivered = "message.delivered"
	EventTypeMessageRead      = "message.read"
	EventTypeMessageCreated   = "message.created"
	EventTypeError            = "error"
	EventTypePresenceChanged  = "presence.changed"
	EventTypeSyncRequired     = "sync.required"
)

const (
	SyncReasonForwardQueueFull = "forward_queue_full"
	SyncReasonPendingQueueFull = "pending_queue_full"
	SyncReasonSendQueueFull    = "send_queue_full"
)
