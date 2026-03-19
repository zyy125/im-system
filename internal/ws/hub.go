package ws

import (
	"context"
	"log"

	"github.com/zyy125/im-system/internal/repository"
)

type ForwardMsg struct {
	To   uint64
	From uint64
	Content []byte
}

type Hub struct {
	Clients    map[uint64]*Client
	Register   chan *Client
	Unregister chan *Client
	Forward    chan *ForwardMsg

	PresenceRepo repository.PresenceRepo
}

func NewHub(presenceRepo repository.PresenceRepo) *Hub {
	return &Hub{
		Clients:      make(map[uint64]*Client),
		Register:     make(chan *Client, 32),
		Unregister:   make(chan *Client, 32),
		Forward:      make(chan *ForwardMsg, 512),
		PresenceRepo: presenceRepo,
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client.UserID] = client

			go func(u uint64) {
				err := h.PresenceRepo.SetOnline(ctx, u)
				if err != nil {
					log.Printf("Set online failed: %v", err)
				}
				log.Printf("Client %d registered, total %d clients", client.UserID, len(h.Clients))
			}(client.UserID)

		case client := <-h.Unregister:
			if _, ok := h.Clients[client.UserID]; ok {
				delete(h.Clients, client.UserID)
				close(client.Send)

				go func(u uint64) {
					err := h.PresenceRepo.SetOffline(ctx, u)
					if err != nil {
						log.Printf("Set offline failed: %v", err)
					}
					log.Printf("Client %d unregistered, total %d clients", client.UserID, len(h.Clients))
				}(client.UserID)
			}

		case msg := <-h.Forward:
			if target, ok := h.Clients[msg.To]; ok {
				select {
				case target.Send <- msg.Content:
				default:
					log.Printf("Forward message to %d failed, target not ready", msg.To)
				}
			} else {
				log.Printf("Forward message to %d failed, target not found", msg.To)
			}

		case <-ctx.Done():
			for uid, client := range h.Clients {
				_ = client.Conn.Close()
				close(client.Send)
				go func(u uint64) {
					err := h.PresenceRepo.SetOffline(context.Background(), u)
					if err != nil {
						log.Printf("Set offline failed: %v", err)
					}
				}(uid)
			}
			log.Printf("Hub context canceled")
			return
		}
	}
}
