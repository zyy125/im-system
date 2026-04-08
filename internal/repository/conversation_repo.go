package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"gorm.io/gorm"
)

type ConversationRepo interface {
	GetOrCreateSingle(ctx context.Context, userA, userB uint64) (model.Conversation, error)
	GetSingle(ctx context.Context, userA, userB uint64) (model.Conversation, error)
	ListMembersByUser(ctx context.Context, userID uint64) ([]model.ConversationMember, error)
	ListConversationsByUser(ctx context.Context, userID uint64) ([]model.Conversation, error)
	GetMember(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error)
	EnsureMember(ctx context.Context, conversationID, userID uint64) error
	SetVisible(ctx context.Context, conversationID, userID uint64, visible bool) error
	UpdateLastDeliveredMsgSeq(ctx context.Context, conversationID, userID, msgSeq uint64) error
	UpdateLastReadMsgSeq(ctx context.Context, conversationID, userID, msgSeq uint64) error
}

type conversationRepo struct {
	db *gorm.DB
}

var _ ConversationRepo = (*conversationRepo)(nil)

func NewConversationRepo(db *gorm.DB) *conversationRepo {
	return &conversationRepo{db: db}
}

func (r *conversationRepo) GetOrCreateSingle(ctx context.Context, userA, userB uint64) (model.Conversation, error) {
	if userA == 0 || userB == 0 {
		return model.Conversation{}, apperr.Required("user_a", "user_b")
	}
	if userA == userB {
		return model.Conversation{}, apperr.FriendCannotAddSelf()
	}

	key := buildSingleKey(userA, userB)
	var conv model.Conversation

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("type = ? AND single_key = ?", model.ConversationTypeSingle, key).First(&conv).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			conv = model.Conversation{
				Type:      model.ConversationTypeSingle,
				SingleKey: stringPtr(key),
			}
			if err := tx.Create(&conv).Error; err != nil {
				if !isDuplicateKeyErr(err) {
					return err
				}
				if err := tx.Where("type = ? AND single_key = ?", model.ConversationTypeSingle, key).First(&conv).Error; err != nil {
					return err
				}
			}
		}

		if err := ensureConversationMember(tx, conv.ID, userA); err != nil {
			return err
		}
		if err := ensureConversationMember(tx, conv.ID, userB); err != nil {
			return err
		}

		return nil
	})

	return conv, err
}

func (r *conversationRepo) GetSingle(ctx context.Context, userA, userB uint64) (model.Conversation, error) {
	if userA == 0 || userB == 0 {
		return model.Conversation{}, apperr.Required("user_a", "user_b")
	}
	if userA == userB {
		return model.Conversation{}, apperr.FriendCannotAddSelf()
	}

	key := buildSingleKey(userA, userB)
	var conv model.Conversation
	err := r.db.WithContext(ctx).
		Where("type = ? AND single_key = ?", model.ConversationTypeSingle, key).
		First(&conv).Error
	return conv, err
}

func (r *conversationRepo) ListMembersByUser(ctx context.Context, userID uint64) ([]model.ConversationMember, error) {
	if userID == 0 {
		return nil, apperr.RequiredOne("user_id")
	}

	var members []model.ConversationMember
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("conversation_id ASC").
		Find(&members).Error
	if err != nil {
		return nil, err
	}

	return members, nil
}

func (r *conversationRepo) ListConversationsByUser(ctx context.Context, userID uint64) ([]model.Conversation, error) {
	if userID == 0 {
		return nil, apperr.RequiredOne("user_id")
	}

	var conversations []model.Conversation
	err := r.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Joins("JOIN conversation_members ON conversation_members.conversation_id = conversations.id").
		Where("conversation_members.user_id = ? AND conversation_members.visible = ?", userID, true).
		Order("conversations.id DESC").
		Find(&conversations).Error
	if err != nil {
		return nil, err
	}

	return conversations, nil
}

func (r *conversationRepo) GetMember(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
	var member model.ConversationMember
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&member).Error
	return member, err
}

func (r *conversationRepo) EnsureMember(ctx context.Context, conversationID, userID uint64) error {
	if conversationID == 0 || userID == 0 {
		return apperr.Required("conversation_id", "user_id")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return ensureConversationMember(tx, conversationID, userID)
	})
}

func (r *conversationRepo) SetVisible(ctx context.Context, conversationID, userID uint64, visible bool) error {
	if conversationID == 0 || userID == 0 {
		return apperr.Required("conversation_id", "user_id")
	}

	result := r.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("visible", visible)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperr.ConversationMemberNotFound()
	}
	return nil
}

func (r *conversationRepo) UpdateLastDeliveredMsgSeq(ctx context.Context, conversationID, userID, msgSeq uint64) error {
	if conversationID == 0 || userID == 0 || msgSeq == 0 {
		return apperr.Required("conversation_id", "user_id", "msg_seq")
	}

	result := r.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ? AND last_delivered_msg_seq < ?", conversationID, userID, msgSeq).
		Update("last_delivered_msg_seq", msgSeq)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		member, err := r.GetMember(ctx, conversationID, userID)
		if err != nil {
			return apperr.ConversationMemberNotFound()
		}
		if member.LastDeliveredMsgSeq >= msgSeq {
			return nil
		}
		return apperr.ConversationMemberUpdateFailed()
	}

	return nil
}

func (r *conversationRepo) UpdateLastReadMsgSeq(ctx context.Context, conversationID, userID, msgSeq uint64) error {
	if conversationID == 0 || userID == 0 || msgSeq == 0 {
		return apperr.Required("conversation_id", "user_id", "msg_seq")
	}

	result := r.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ? AND last_read_msg_seq < ?", conversationID, userID, msgSeq).
		Update("last_read_msg_seq", msgSeq)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		member, err := r.GetMember(ctx, conversationID, userID)
		if err != nil {
			return err
		}
		if member.LastReadMsgSeq >= msgSeq {
			return nil
		}
		return apperr.ConversationMemberUpdateFailed()
	}
	return nil
}

func ensureConversationMember(tx *gorm.DB, conversationID, userID uint64) error {
	var member model.ConversationMember
	err := tx.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&member).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	member = model.ConversationMember{
		ConversationID: conversationID,
		UserID:         userID,
		Role:           model.ConversationMemberRoleMember,
		Visible:        true,
	}
	return tx.Create(&member).Error
}

func buildSingleKey(userA, userB uint64) string {
	if userA > userB {
		userA, userB = userB, userA
	}
	return fmt.Sprintf("%d:%d", userA, userB)
}

func stringPtr(value string) *string {
	return &value
}
