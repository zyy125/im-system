package ws

import (
	"context"
	"sync"

	"github.com/zyy125/im-system/internal/logging"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
)

type ForwardMessage struct {
	To      uint64
	From    uint64
	Content []byte
}

type ClientBootstrapResult struct {
	Client          *Client
	OfflineMessages [][]byte
}

type PresenceAudienceProvider interface {
	ListFriendIDs(ctx context.Context, userID uint64) ([]uint64, error)
}

type Hub struct {
	Clients            map[uint64]*Client
	ReadyClients       map[uint64]bool
	NeedsSync          map[uint64]bool
	PendingMessages    map[uint64][][]byte
	Register           chan *Client
	Unregister         chan *Client
	Forward            chan *ForwardMessage
	LifecycleForward   chan *ForwardMessage
	ClientBootstrapped chan *ClientBootstrapResult

	Lifecycle ClientLifecycle
	mu        sync.RWMutex
}

type OfflineMessageLoader interface {
	ListOfflineMessages(ctx context.Context, userID uint64) ([]model.Message, error)
}

func NewHub(
	presenceRepo repository.PresenceRepo,
	offlineLoader OfflineMessageLoader,
	presenceAudience PresenceAudienceProvider,
) *Hub {
	lifecycleForward := make(chan *ForwardMessage, 256)
	return &Hub{
		Clients:            make(map[uint64]*Client),
		ReadyClients:       make(map[uint64]bool),
		NeedsSync:          make(map[uint64]bool),
		PendingMessages:    make(map[uint64][][]byte),
		Register:           make(chan *Client, 32),
		Unregister:         make(chan *Client, 32),
		Forward:            make(chan *ForwardMessage, 512),
		LifecycleForward:   lifecycleForward,
		ClientBootstrapped: make(chan *ClientBootstrapResult, 32),
		Lifecycle:          NewClientLifecycle(presenceRepo, offlineLoader, presenceAudience, lifecycleForward),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client.UserID] = client
			h.ReadyClients[client.UserID] = false
			delete(h.PendingMessages, client.UserID)
			client.Lifecycle = h.Lifecycle
			go h.bootstrapClient(ctx, client)
			logging.With("user_id", client.UserID, "connection_id", client.ConnectionID).Info("client registered", "client_count", len(h.Clients))

		case client := <-h.Unregister:
			current, ok := h.Clients[client.UserID]
			if !ok || current != client {
				continue
			}
			delete(h.Clients, client.UserID)
			delete(h.ReadyClients, client.UserID)
			close(client.Send)

			go h.disconnectClient(ctx, client.UserID)
			logging.With("user_id", client.UserID, "connection_id", client.ConnectionID).Info("client unregistered", "client_count", len(h.Clients))

		case msg := <-h.Forward:
			h.forwardLocal(msg)

		case msg := <-h.LifecycleForward:
			h.forwardLocal(msg)

		case result := <-h.ClientBootstrapped:
			current, ok := h.Clients[result.Client.UserID]
			if !ok || current != result.Client {
				continue
			}
			h.flushMessages(current, result.OfflineMessages)
			h.flushMessages(current, h.PendingMessages[current.UserID])
			delete(h.PendingMessages, current.UserID)
			delete(h.NeedsSync, current.UserID)
			h.ReadyClients[current.UserID] = true

		case <-ctx.Done():
			h.CloseAll()
			for uid := range h.Clients {
				go func(u uint64) {
					h.disconnectClient(context.Background(), u)
				}(uid)
			}
			logging.With("event_type", "hub_shutdown").Info("hub context canceled")
			return
		}
	}
}

func (h *Hub) forwardLocal(msg *ForwardMessage) bool {
	target, ok := h.Clients[msg.To]
	if !ok {
		return false
	}
	if !h.ReadyClients[msg.To] {
		h.enqueuePending(msg.To, msg.Content)
		return true
	}
	h.trySend(target, msg.Content)
	return true
}

func closeClientConn(client *Client) {
	if client == nil || client.Conn == nil {
		return
	}
	_ = client.Conn.Close()
}

func (h *Hub) bootstrapClient(ctx context.Context, client *Client) {
	payloads := make([][]byte, 0)
	if h.Lifecycle != nil {
		offlinePayloads, err := h.Lifecycle.Bootstrap(ctx, client.UserID)
		if err != nil {
			logging.With("user_id", client.UserID, "connection_id", client.ConnectionID).Error("bootstrap client failed", "error", err)
		} else {
			payloads = offlinePayloads
		}
	}
	select {
	case h.ClientBootstrapped <- &ClientBootstrapResult{
		Client:          client,
		OfflineMessages: payloads,
	}:
	case <-ctx.Done():
	}
}

func (h *Hub) disconnectClient(ctx context.Context, userID uint64) {
	if h.Lifecycle != nil {
		h.Lifecycle.Disconnect(ctx, userID)
	}
}

func (h *Hub) enqueuePending(userID uint64, payload []byte) {
	const maxPendingPerUser = 512

	queue := h.PendingMessages[userID]
	if len(queue) >= maxPendingPerUser {
		h.MarkNeedsSync(userID)
		logging.With("user_id", userID).Warn("pending queue is full, mark connection needs sync")
		return
	}
	h.PendingMessages[userID] = append(queue, payload)
}

func (h *Hub) flushMessages(client *Client, payloads [][]byte) {
	for _, payload := range payloads {
		h.trySend(client, payload)
	}
}

func (h *Hub) trySend(client *Client, payload []byte) {
	select {
	case client.Send <- payload:
	default:
		h.MarkNeedsSync(client.UserID)
		logging.With("user_id", client.UserID, "connection_id", client.ConnectionID).Warn("send queue is full, mark connection needs sync")
	}
}

func (h *Hub) MarkNeedsSync(userID uint64) {
	h.NeedsSync[userID] = true
}

func (h *Hub) CloseAll() {
	for uid, client := range h.Clients {
		closeClientConn(client)
		close(client.Send)
		delete(h.ReadyClients, uid)
		delete(h.PendingMessages, uid)
	}
}
