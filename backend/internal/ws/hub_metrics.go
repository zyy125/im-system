package ws

import "sync/atomic"

type HubMetrics struct {
	users                      atomic.Int64
	connections                atomic.Int64
	registerTotal              atomic.Int64
	unregisterTotal            atomic.Int64
	forwardQueueFullTotal      atomic.Int64
	pendingQueueFullTotal      atomic.Int64
	sendQueueFullTotal         atomic.Int64
	syncRequiredEmittedTotal   atomic.Int64
	bootstrapTotal             atomic.Int64
	bootstrapFailedTotal       atomic.Int64
	syncDeliveryFailCloseTotal atomic.Int64
}

type HubSnapshot struct {
	Users                int64 `json:"users"`
	Connections          int64 `json:"connections"`
	RegisterQueueLen     int   `json:"register_queue_len"`
	RegisterQueueCap     int   `json:"register_queue_cap"`
	UnregisterQueueLen   int   `json:"unregister_queue_len"`
	UnregisterQueueCap   int   `json:"unregister_queue_cap"`
	ForwardQueueLen      int   `json:"forward_queue_len"`
	ForwardQueueCap      int   `json:"forward_queue_cap"`
	LifecycleForwardLen  int   `json:"lifecycle_forward_len"`
	LifecycleForwardCap  int   `json:"lifecycle_forward_cap"`
	MarkSyncQueueLen     int   `json:"mark_sync_queue_len"`
	MarkSyncQueueCap     int   `json:"mark_sync_queue_cap"`
	BootstrappedQueueLen int   `json:"bootstrapped_queue_len"`
	BootstrappedQueueCap int   `json:"bootstrapped_queue_cap"`

	RegisterTotal              int64 `json:"register_total"`
	UnregisterTotal            int64 `json:"unregister_total"`
	ForwardQueueFullTotal      int64 `json:"forward_queue_full_total"`
	PendingQueueFullTotal      int64 `json:"pending_queue_full_total"`
	SendQueueFullTotal         int64 `json:"send_queue_full_total"`
	SyncRequiredEmittedTotal   int64 `json:"sync_required_emitted_total"`
	BootstrapTotal             int64 `json:"bootstrap_total"`
	BootstrapFailedTotal       int64 `json:"bootstrap_failed_total"`
	SyncDeliveryFailCloseTotal int64 `json:"sync_delivery_fail_close_total"`
}
