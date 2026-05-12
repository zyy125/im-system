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

		recipients = recipients[:0]
		for _, current := range members {
			recipients = append(recipients, current.UserID)
			if err := conversationRepo.SetVisible(ctx, conversationID, current.UserID, true); err != nil {
				return err
			}
		}
		if err := conversationRepo.UpdateLastAckedMsgSeq(ctx, conversationID, senderID, msg.Seq); err != nil && apperr.CodeOf(err) != apperr.CodeConversationMemberNotFound {
			return err
		}
		if err := conversationRepo.UpdateLastReadMsgSeq(ctx, conversationID, senderID, msg.Seq); err != nil && apperr.CodeOf(err) != apperr.CodeConversationMemberNotFound {
			return err
		}
		return nil
	}); err != nil {
		return model.Message{}, nil, err
	}

	return msg, recipients, nil
}
