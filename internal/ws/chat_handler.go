package ws

import (
	"context"
	"time"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/service"
)

type ChatSendHandler interface {
	HandleChatSend(ctx context.Context, senderID uint64, req ClientChatSend) (*ForwardMessage, error)
}

type FriendChecker interface {
	AreFriends(ctx context.Context, userID, friendID uint64) (bool, error)
}

type ConversationIDProvider interface {
	EnsureDirectConversationID(ctx context.Context, userA, userB uint64) (string, error)
}

type chatSendHandler struct {
	messageService       service.MessageService
	friendChecker        FriendChecker
	conversationProvider ConversationIDProvider
}

func NewChatSendHandler(
	messageService service.MessageService,
	friendChecker FriendChecker,
	conversationProvider ConversationIDProvider,
) ChatSendHandler {
	return &chatSendHandler{
		messageService:       messageService,
		friendChecker:        friendChecker,
		conversationProvider: conversationProvider,
	}
}

func (h *chatSendHandler) HandleChatSend(ctx context.Context, senderID uint64, req ClientChatSend) (*ForwardMessage, error) {
	if req.MsgID == "" || req.To == 0 || req.Content == "" {
		return nil, apperr.MessageInvalidPayload()
	}

	if h.friendChecker != nil {
		ok, err := h.friendChecker.AreFriends(ctx, senderID, req.To)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apperr.FriendNotFriends()
		}
	}

	chatMsg := model.ChatMessage{
		MsgID:    req.MsgID,
		From:     senderID,
		To:       req.To,
		Content:  req.Content,
		SendTime: req.SendTime,
	}

	if h.conversationProvider == nil {
		return nil, apperr.Internal("conversation provider unavailable", nil)
	}

	conversationID, err := h.conversationProvider.EnsureDirectConversationID(ctx, senderID, req.To)
	if err != nil {
		return nil, err
	}
	chatMsg.ConversationID = conversationID

	if chatMsg.SendTime == 0 {
		chatMsg.SendTime = time.Now().UnixMilli()
	}
	if h.messageService == nil {
		return nil, apperr.Internal("message service unavailable", nil)
	}

	saved, err := h.messageService.SaveMessage(ctx, &chatMsg)
	if err != nil {
		return nil, err
	}

	realtimePayload, err := MarshalEnvelope(EventTypeChatMessage, NewServerChatMessage(saved))
	if err != nil {
		return nil, err
	}

	return &ForwardMessage{
		To:      saved.To,
		From:    senderID,
		Content: realtimePayload,
	}, nil
}
