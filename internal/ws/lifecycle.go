package ws

import (
	"context"
	"log"

	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
)

type ClientLifecycle interface {
	Bootstrap(ctx context.Context, userID uint64) ([][]byte, error)
	Disconnect(ctx context.Context, userID uint64)
}

type clientLifecycle struct {
	presenceRepo     repository.PresenceRepo
	offlineLoader    OfflineMessageLoader
	presenceAudience PresenceAudienceProvider
	forward          chan<- *ForwardMessage
}

func NewClientLifecycle(
	presenceRepo repository.PresenceRepo,
	offlineLoader OfflineMessageLoader,
	presenceAudience PresenceAudienceProvider,
	forward chan<- *ForwardMessage,
) ClientLifecycle {
	return &clientLifecycle{
		presenceRepo:     presenceRepo,
		offlineLoader:    offlineLoader,
		presenceAudience: presenceAudience,
		forward:          forward,
	}
}

func (l *clientLifecycle) Bootstrap(ctx context.Context, userID uint64) ([][]byte, error) {
	if l.presenceRepo != nil {
		if err := l.presenceRepo.SetOnline(ctx, userID); err != nil {
			log.Printf("Set online failed for %d: %v", userID, err)
		} else {
			l.broadcastPresence(ctx, userID, true)
		}
	}

	msgs, err := l.loadOfflineMessages(ctx, userID)
	if err != nil {
		return nil, err
	}

	payloads := make([][]byte, 0, len(msgs))
	for _, msg := range msgs {
		payload, err := MarshalEnvelope(EventTypeChatMessage, NewServerChatMessage(msg))
		if err != nil {
			log.Printf("Marshal offline message %s failed: %v", msg.MsgID, err)
			continue
		}
		payloads = append(payloads, payload)
	}
	return payloads, nil
}

func (l *clientLifecycle) Disconnect(ctx context.Context, userID uint64) {
	if l.presenceRepo == nil {
		return
	}
	if err := l.presenceRepo.SetOffline(ctx, userID); err != nil {
		log.Printf("Set offline failed for %d: %v", userID, err)
		return
	}
	l.broadcastPresence(ctx, userID, false)
}

func (l *clientLifecycle) loadOfflineMessages(ctx context.Context, userID uint64) ([]model.ChatMessage, error) {
	if l.offlineLoader == nil {
		return []model.ChatMessage{}, nil
	}
	msgs, err := l.offlineLoader.ListOfflineMessages(ctx, userID)
	if err != nil {
		log.Printf("Load offline messages for %d failed: %v", userID, err)
		return nil, err
	}
	return msgs, nil
}

func (l *clientLifecycle) broadcastPresence(ctx context.Context, userID uint64, online bool) {
	if l.presenceAudience == nil || l.forward == nil {
		return
	}

	friendIDs, err := l.presenceAudience.ListFriendIDs(ctx, userID)
	if err != nil {
		log.Printf("List presence audience for %d failed: %v", userID, err)
		return
	}
	if len(friendIDs) == 0 {
		return
	}

	payload, err := MarshalEnvelope(EventTypePresenceChanged, PresenceChangedData{
		UserID: userID,
		Online: online,
	})
	if err != nil {
		log.Printf("Marshal presence event for %d failed: %v", userID, err)
		return
	}

	for _, friendID := range friendIDs {
		select {
		case l.forward <- &ForwardMessage{
			To:      friendID,
			Content: payload,
		}:
		default:
			log.Printf("Presence forward queue is full for user %d", friendID)
		}
	}
}
