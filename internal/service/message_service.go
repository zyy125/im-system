package service

import (
	"context"
	"strconv"
	"time"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
)

type messageService struct {
	messageRepo      repository.MessageRepo
	conversationRepo repository.ConversationRepo
	txManager        repository.MessageTxManager
}

type MessageService interface {
	SaveMessage(ctx context.Context, msg *model.ChatMessage) (model.ChatMessage, error)
	ListHistory(ctx context.Context, userID, peerID uint64, limit int, beforeID uint64) ([]model.ChatMessage, bool, error)
}

var _ MessageService = (*messageService)(nil)

func NewMessageService(messageRepo repository.MessageRepo, conversationRepo repository.ConversationRepo, txManager repository.MessageTxManager) MessageService {
	return &messageService{messageRepo: messageRepo, conversationRepo: conversationRepo, txManager: txManager}
}

func (s *messageService) SaveMessage(ctx context.Context, msg *model.ChatMessage) (model.ChatMessage, error) {
	if msg == nil {
		return model.ChatMessage{}, apperr.RequiredOne("message")
	}
	if msg.MsgID == "" {
		return model.ChatMessage{}, apperr.MessageIDRequired()
	}
	if msg.ConversationID == "" {
		return model.ChatMessage{}, apperr.MessageConversationRequired()
	}
	if msg.From == 0 || msg.To == 0 {
		return model.ChatMessage{}, apperr.Required("from", "to")
	}
	if msg.Content == "" {
		return model.ChatMessage{}, apperr.RequiredOne("content")
	}

	conversationID, err := strconv.ParseUint(msg.ConversationID, 10, 64)
	if err != nil {
		return model.ChatMessage{}, apperr.InvalidID("conversation_id")
	}
	if msg.SendTime == 0 {
		msg.SendTime = time.Now().UnixMilli()
	}

	persist := func(messageRepo repository.MessageRepo, conversationRepo repository.ConversationRepo) error {
		if err := messageRepo.Create(ctx, msg); err != nil {
			return err
		}

		if err := conversationRepo.EnsureMember(ctx, conversationID, msg.From); err != nil {
			return err
		}
		if err := conversationRepo.EnsureMember(ctx, conversationID, msg.To); err != nil {
			return err
		}

		if err := conversationRepo.UpdateLastDeliveredMsgSeq(ctx, conversationID, msg.To, msg.ID); err != nil {
			return err
		}
		if err := conversationRepo.SetVisible(ctx, conversationID, msg.From, true); err != nil {
			return err
		}
		return conversationRepo.SetVisible(ctx, conversationID, msg.To, true)
	}

	if s.txManager != nil {
		if err := s.txManager.WithinMessageTx(ctx, persist); err != nil {
			return model.ChatMessage{}, err
		}
	} else if err := persist(s.messageRepo, s.conversationRepo); err != nil {
		return model.ChatMessage{}, err
	}

	return *msg, nil
}

func (s *messageService) ListHistory(ctx context.Context, userID, peerID uint64, limit int, beforeID uint64) ([]model.ChatMessage, bool, error) {
	if userID == 0 || peerID == 0 {
		return nil, false, apperr.Required("user_id", "peer_id")
	}
	return s.messageRepo.ListBetween(ctx, userID, peerID, limit, beforeID)
}
