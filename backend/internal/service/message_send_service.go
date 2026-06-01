package service

import (
	"context"
	"time"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
)

type MessageSendService interface {
	SendTextMessage(ctx context.Context, senderID, conversationID uint64, msgID, content string) (model.Message, []uint64, error)
}

type messageSendService struct {
	txManager    repository.MessageTxManager
	seqAllocator SeqAllocator
	friendRepo   repository.FriendRepo
}

func NewMessageSendService(
	txManager repository.MessageTxManager,
	seqAllocator SeqAllocator,
) MessageSendService {
	return &messageSendService{
		txManager:    txManager,
		seqAllocator: seqAllocator,
	}
}

func NewMessageSendServiceWithFriendRepo(
	txManager repository.MessageTxManager,
	seqAllocator SeqAllocator,
	friendRepo repository.FriendRepo,
) MessageSendService {
	return &messageSendService{
		txManager:    txManager,
		seqAllocator: seqAllocator,
		friendRepo:   friendRepo,
	}
}

func (s *messageSendService) SendTextMessage(ctx context.Context, senderID, conversationID uint64, msgID, content string) (model.Message, []uint64, error) {
	if senderID == 0 || conversationID == 0 || msgID == "" || content == "" {
		return model.Message{}, nil, apperr.MessageInvalidPayload()
	}
	if s.seqAllocator == nil || s.txManager == nil {
		return model.Message{}, nil, apperr.Internal("message send service unavailable", nil)
	}

	seq, err := s.seqAllocator.Allocate(ctx, conversationID)
	if err != nil {
		return model.Message{}, nil, err
	}

	msg := model.Message{
		MsgID:          msgID,
		ConversationID: conversationID,
		Seq:            seq,
		Type:           model.MessageTypeText,
		Event:          model.MessageEventNone,
		From:           senderID,
		SendTime:       time.Now().UnixMilli(),
		Content:        content,
	}
	recipients := make([]uint64, 0)

	if err := s.txManager.WithinMessageTx(ctx, func(messageRepo repository.MessageRepo, conversationRepo repository.ConversationRepo) error {
		conv, err := conversationRepo.GetByID(ctx, conversationID)
		if err != nil {
			return err
		}
		if !conv.IsActive() {
			return apperr.ConversationDismissed()
		}
		if err := ensureSingleConversationFriendship(ctx, s.friendRepo, conv, senderID); err != nil {
			return err
		}

		member, err := conversationRepo.GetMember(ctx, conversationID, senderID)
		if err != nil {
			return err
		}
		if !member.IsActive() {
			return apperr.ConversationNotAccessible()
		}

		members, err := conversationRepo.ListActiveMembers(ctx, conversationID)
		if err != nil {
			return err
		}
		if len(members) == 0 {
			return apperr.ConversationMemberNotFound()
		}

		if err := messageRepo.Create(ctx, &msg); err != nil {
			return err
		}

		recipients = memberUserIDs(members)
		if err := conversationRepo.SetVisibleForUsers(ctx, conversationID, recipients, true); err != nil {
			return err
		}
		if err := conversationRepo.AdvanceSenderMessageCursors(ctx, conversationID, senderID, msg.Seq); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return model.Message{}, nil, err
	}

	return msg, recipients, nil
}
