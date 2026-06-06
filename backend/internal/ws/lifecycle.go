package ws

import (
	"context"

	"github.com/zyy125/im-system/internal/logging"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/internal/service"
)

type ClientLifecycle interface {
	Connect(ctx context.Context, userID uint64)
	Bootstrap(ctx context.Context, userID uint64) ([][]byte, error)
	Refresh(ctx context.Context, userID uint64)
	Disconnect(ctx context.Context, userID uint64)
}

type clientLifecycle struct {
	presenceRepo        repository.PresenceRepo
	conversationService service.ConversationService
	friendRepo          repository.FriendRepo
	userService         service.UserService
	hub                 *Hub
}

func NewClientLifecycle(
	presenceRepo repository.PresenceRepo,
	conversationService service.ConversationService,
	friendRepo repository.FriendRepo,
	userService service.UserService,
	hub *Hub,
) ClientLifecycle {
	return &clientLifecycle{
		presenceRepo:        presenceRepo,
		conversationService: conversationService,
		friendRepo:          friendRepo,
		userService:         userService,
		hub:                 hub,
	}
}

func (l *clientLifecycle) Connect(ctx context.Context, userID uint64) {
	logger := logging.FromContext(ctx).With("user_id", userID)
	if l.presenceRepo != nil {
		if err := l.presenceRepo.SetOnline(ctx, userID); err != nil {
			logger.Error("set online failed", "error", err)
		} else {
			l.broadcastPresence(ctx, userID, true)
		}
	}
}

func (l *clientLifecycle) Bootstrap(ctx context.Context, userID uint64) ([][]byte, error) {
	logger := logging.FromContext(ctx).With("user_id", userID)
	msgs, err := l.loadOfflineMessages(ctx, userID)
	if err != nil {
		return nil, err
	}

	payloads := make([][]byte, 0, len(msgs))
	for _, msg := range msgs {
		payload, err := MarshalEnvelope(EventTypeMessageCreated, NewServerMessage(msg))
		if err != nil {
			logger.Error("marshal offline message failed", "msg_id", msg.MsgID, "error", err)
			continue
		}
		payloads = append(payloads, payload)
	}
	return payloads, nil
}

func (l *clientLifecycle) Refresh(ctx context.Context, userID uint64) {
	if l.presenceRepo == nil {
		return
	}
	if err := l.presenceRepo.RefreshOnline(ctx, userID); err != nil {
		logging.FromContext(ctx).With("user_id", userID).Error("refresh presence failed", "error", err)
	}
}

func (l *clientLifecycle) Disconnect(ctx context.Context, userID uint64) {
	if l.presenceRepo == nil {
		return
	}
	if err := l.presenceRepo.SetOffline(ctx, userID); err != nil {
		logging.FromContext(ctx).With("user_id", userID).Error("set offline failed", "error", err)
		return
	}
	l.broadcastPresence(ctx, userID, false)
}

func (l *clientLifecycle) loadOfflineMessages(ctx context.Context, userID uint64) ([]model.Message, error) {
	if l.conversationService == nil {
		return []model.Message{}, nil
	}
	msgs, err := l.conversationService.ListOfflineMessages(ctx, userID)
	if err != nil {
		logging.FromContext(ctx).With("user_id", userID).Error("load offline messages failed", "error", err)
		return nil, err
	}
	return msgs, nil
}

func (l *clientLifecycle) broadcastPresence(ctx context.Context, userID uint64, online bool) {
	if l.friendRepo == nil || l.hub == nil || l.hub.LifecycleForward == nil {
		return
	}

	friendIDs, err := l.friendRepo.ListFriendIDs(ctx, userID)
	if err != nil {
		logging.FromContext(ctx).With("user_id", userID).Error("list presence audience failed", "error", err)
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
		logging.FromContext(ctx).With("user_id", userID).Error("marshal presence event failed", "error", err)
		return
	}

	for _, friendID := range friendIDs {
		select {
		case l.hub.LifecycleForward <- &ForwardMessage{
			To:      friendID,
			Content: payload,
		}:
		default:
			l.hub.EnqueueUserSync(friendID, 0, SyncReasonForwardQueueFull)
			logging.FromContext(ctx).With("user_id", userID, "target_user_id", friendID).Warn("presence forward queue is full")
		}
	}
}

