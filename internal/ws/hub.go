package ws

import (
	"context"
	"log"

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
	PendingMessages    map[uint64][][]byte
	Register           chan *Client
	Unregister         chan *Client
	Forward            chan *ForwardMessage
	LifecycleForward   chan *ForwardMessage
	ClientBootstrapped chan *ClientBootstrapResult

	Lifecycle ClientLifecycle
}

type OfflineMessageLoader interface {
	ListOfflineMessages(ctx context.Context, userID uint64) ([]model.ChatMessage, error)
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
			go h.bootstrapClient(ctx, client)
			log.Printf("Client %d registered, total %d clients", client.UserID, len(h.Clients))

		case client := <-h.Unregister:
			current, ok := h.Clients[client.UserID]
			/*判断 current == client。
			这是为了避免“旧连接断开，把新连接误删”。
			比如：
			用户先有旧连接 A
			很快又重连出新连接 B
			旧连接 A 这时才触发 Unregister
			*/
			if !ok || current != client {
				continue
			}
			delete(h.Clients, client.UserID)
			delete(h.ReadyClients, client.UserID)
			delete(h.PendingMessages, client.UserID)
			close(client.Send)

			go h.disconnectClient(ctx, client.UserID)
			log.Printf("Client %d unregistered, total %d clients", client.UserID, len(h.Clients))

		case msg := <-h.Forward:
			target, ok := h.Clients[msg.To]
			if !ok {
				log.Printf("User %d is offline, realtime forward skipped and offline sync will rely on persisted messages", msg.To)
				continue
			}
			if !h.ReadyClients[msg.To] {
				h.enqueuePending(msg.To, msg.Content)
				continue
			}
			h.trySend(target, msg.Content)

		case msg := <-h.LifecycleForward:
			target, ok := h.Clients[msg.To]
			if !ok {
				continue
			}
			if !h.ReadyClients[msg.To] {
				h.enqueuePending(msg.To, msg.Content)
				continue
			}
			h.trySend(target, msg.Content)

		case result := <-h.ClientBootstrapped:
			current, ok := h.Clients[result.Client.UserID]
			if !ok || current != result.Client {
				continue
			}
			h.flushMessages(current, result.OfflineMessages)
			h.flushMessages(current, h.PendingMessages[current.UserID])
			delete(h.PendingMessages, current.UserID)
			h.ReadyClients[current.UserID] = true

		case <-ctx.Done():
			for uid, client := range h.Clients {
				closeClientConn(client)
				close(client.Send)
				delete(h.ReadyClients, uid)
				delete(h.PendingMessages, uid)
				go func(u uint64) {
					h.disconnectClient(context.Background(), u)
				}(uid)
			}
			log.Printf("Hub context canceled")
			return
		}
	}
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
			log.Printf("Bootstrap client %d failed: %v", client.UserID, err)
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
		log.Printf("Pending queue for user %d is full, dropping message", userID)
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
		log.Printf("Send message to %d failed, target not ready", client.UserID)
	}
}
