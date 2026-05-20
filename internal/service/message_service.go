package service

import (
	"context"
	"errors"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
	"gorm.io/gorm"
)

type messageService struct {
	messageRepo      repository.MessageRepo
	conversationRepo repository.ConversationRepo
}

type MessageService interface {
	// ListConversationHistory 按 seq 分页查询会话历史消息，并校验当前用户访问权限。
	ListConversationHistory(ctx context.Context, userID, conversationID uint64, limit int, beforeSeq uint64) ([]model.Message, bool, error)
	// SyncConversation 按 seq 补拉某个会话 after_seq 之后的消息。
	SyncConversation(ctx context.Context, userID, conversationID, afterSeq uint64, limit int) ([]model.Message, bool, error)
	// MarkDelivered 推进某个成员的最大已确认接收 seq，并返回需要接收回执的活跃成员。
	MarkDelivered(ctx context.Context, userID, conversationID, deliveredSeq uint64) ([]uint64, error)
}

var _ MessageService = (*messageService)(nil)

// NewMessageService 创建一个不依赖运行时热状态仓库的消息服务。
// 适用于单元测试或只关心持久化行为的调用方。
func NewMessageService(
	messageRepo repository.MessageRepo,
	conversationRepo repository.ConversationRepo,
) MessageService {
	return &messageService{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
	}
}

func (s *messageService) ListConversationHistory(ctx context.Context, userID, conversationID uint64, limit int, beforeSeq uint64) ([]model.Message, bool, error) {
	if userID == 0 || conversationID == 0 {
		return nil, false, apperr.Required("user_id", "conversation_id")
	}

	_, member, err := s.requireActiveConversationMember(ctx, conversationID, userID)
	if err != nil {
		return nil, false, err
	}

	return s.messageRepo.ListConversationHistory(ctx, conversationID, limit, beforeSeq, member.JoinedMsgSeq)
}

func (s *messageService) SyncConversation(ctx context.Context, userID, conversationID, afterSeq uint64, limit int) ([]model.Message, bool, error) {
	if userID == 0 || conversationID == 0 {
		return nil, false, apperr.Required("user_id", "conversation_id")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}

	_, member, err := s.requireActiveConversationMember(ctx, conversationID, userID)
	if err != nil {
		return nil, false, err
	}

	if afterSeq < member.JoinedMsgSeq {
		afterSeq = member.JoinedMsgSeq
	}

	return s.messageRepo.ListConversationAfterSeq(ctx, conversationID, afterSeq, limit)
}

func (s *messageService) MarkDelivered(ctx context.Context, userID, conversationID, deliveredSeq uint64) ([]uint64, error) {
	if userID == 0 || conversationID == 0 || deliveredSeq == 0 {
		return nil, apperr.Required("user_id", "conversation_id", "delivered_seq")
	}

	_, member, err := s.requireActiveConversationMember(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	if deliveredSeq <= member.JoinedMsgSeq || deliveredSeq <= member.LastAckedMsgSeq {
		recipients, err := s.listActiveMemberIDs(ctx, conversationID)
		if err != nil {
			return nil, err
		}
		return recipients, nil
	}

	upperBound, err := s.getAckUpperBound(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if deliveredSeq > upperBound {
		return nil, apperr.MessageNotDelivered()
	}

	if err := s.conversationRepo.UpdateLastAckedMsgSeq(ctx, conversationID, userID, deliveredSeq); err != nil {
		return nil, err
	}

	return s.listActiveMemberIDs(ctx, conversationID)
}

func (s *messageService) getAckUpperBound(ctx context.Context, conversationID uint64) (uint64, error) {
	return s.messageRepo.GetMaxSeqByConversation(ctx, conversationID)
}

func (s *messageService) requireActiveConversationMember(ctx context.Context, conversationID, userID uint64) (model.Conversation, model.ConversationMember, error) {
	conv, err := s.conversationRepo.GetByID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Conversation{}, model.ConversationMember{}, apperr.NotFound(apperr.CodeConversationNotFound, "conversation not found")
		}
		return model.Conversation{}, model.ConversationMember{}, err
	}
	if !conv.IsActive() {
		return model.Conversation{}, model.ConversationMember{}, apperr.ConversationDismissed()
	}

	member, err := s.conversationRepo.GetMember(ctx, conversationID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Conversation{}, model.ConversationMember{}, apperr.ConversationMemberNotFound()
		}
		return model.Conversation{}, model.ConversationMember{}, err
	}
	if !member.IsActive() {
		return model.Conversation{}, model.ConversationMember{}, apperr.ConversationNotAccessible()
	}
	return conv, member, nil
}

func (s *messageService) listActiveMemberIDs(ctx context.Context, conversationID uint64) ([]uint64, error) {
	members, err := s.conversationRepo.ListActiveMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	return memberUserIDs(members), nil
}
