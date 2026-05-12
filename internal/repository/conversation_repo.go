package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ConversationRepo interface {
	// Create 插入一条新的会话记录。
	Create(ctx context.Context, conversation *model.Conversation) error
	// GetByID 按主键查询会话。
	GetByID(ctx context.Context, conversationID uint64) (model.Conversation, error)
	// GetOrCreateSingle 获取或创建唯一单聊会话，并确保双方成员存在。
	GetOrCreateSingle(ctx context.Context, userA, userB uint64) (model.Conversation, error)
	// ListMembersByUser 查询某个用户加入过的全部会话成员记录。
	ListMembersByUser(ctx context.Context, userID uint64) ([]model.ConversationMember, error)
	// ListConversationsByUser 查询某个用户当前可见且仍活跃的会话。
	ListConversationsByUser(ctx context.Context, userID uint64) ([]model.Conversation, error)
	// ListActiveGroupsByUser 查询某个用户当前仍为活跃成员的全部群聊，不受 visible 影响。
	ListActiveGroupsByUser(ctx context.Context, userID uint64) ([]model.Conversation, error)
	// ListActiveMembers 查询某个会话下当前仍有效的成员。
	ListActiveMembers(ctx context.Context, conversationID uint64) ([]model.ConversationMember, error)
	// CountActiveMembers 统计某个会话当前活跃成员数。
	CountActiveMembers(ctx context.Context, conversationID uint64) (int64, error)
	// GetMember 查询某个用户在指定会话中的成员记录。
	GetMember(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error)
	// UpsertMember 按会话和用户维度插入或更新成员记录。
	UpsertMember(ctx context.Context, member *model.ConversationMember) error
	// SetVisible 修改某个成员对会话的显示状态。
	SetVisible(ctx context.Context, conversationID, userID uint64, visible bool) error
	// UpdateName 修改会话名称。
	UpdateName(ctx context.Context, conversationID uint64, name string) error
	// UpdateStatus 修改会话整体状态，例如解散。
	UpdateStatus(ctx context.Context, conversationID uint64, status model.ConversationStatus) error
	// UpdateMemberStatus 修改某个成员在会话中的状态和可见性。
	UpdateMemberStatus(ctx context.Context, conversationID, userID uint64, status model.ConversationMemberStatus, visible bool) error
	// UpdateAllMemberStatus 批量修改一个会话下全部成员的状态和可见性。
	UpdateAllMemberStatus(ctx context.Context, conversationID uint64, status model.ConversationMemberStatus, visible bool) error
	// UpdateLastAckedMsgSeq 推进某个成员的接收 ACK 游标。
	UpdateLastAckedMsgSeq(ctx context.Context, conversationID, userID, msgSeq uint64) error
	// UpdateLastReadMsgSeq 推进某个成员的已读游标。
	UpdateLastReadMsgSeq(ctx context.Context, conversationID, userID, msgSeq uint64) error
}

type conversationRepo struct {
	db *gorm.DB
}

var _ ConversationRepo = (*conversationRepo)(nil)

func NewConversationRepo(db *gorm.DB) *conversationRepo {
	return &conversationRepo{db: db}
}

func (r *conversationRepo) Create(ctx context.Context, conversation *model.Conversation) error {
	if conversation == nil {
		return apperr.RequiredOne("conversation")
	}
	return r.db.WithContext(ctx).Create(conversation).Error
}

func (r *conversationRepo) GetByID(ctx context.Context, conversationID uint64) (model.Conversation, error) {
	if conversationID == 0 {
		return model.Conversation{}, apperr.RequiredOne("conversation_id")
	}
	var conv model.Conversation
	err := r.db.WithContext(ctx).First(&conv, conversationID).Error
	return conv, err
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
				Status:    model.ConversationStatusActive,
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
		Where("conversation_members.user_id = ? AND conversation_members.visible = ? AND conversation_members.status = ? AND conversations.status = ?", userID, true, model.ConversationMemberStatusActive, model.ConversationStatusActive).
		Order("conversations.id DESC").
		Find(&conversations).Error
	if err != nil {
		return nil, err
	}

	return conversations, nil
}

func (r *conversationRepo) ListActiveGroupsByUser(ctx context.Context, userID uint64) ([]model.Conversation, error) {
	if userID == 0 {
		return nil, apperr.RequiredOne("user_id")
	}

	var conversations []model.Conversation
	err := r.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Joins("JOIN conversation_members ON conversation_members.conversation_id = conversations.id").
		Where(
			"conversation_members.user_id = ? AND conversation_members.status = ? AND conversations.type = ? AND conversations.status = ?",
			userID,
			model.ConversationMemberStatusActive,
			model.ConversationTypeGroup,
			model.ConversationStatusActive,
		).
		Order("conversations.id DESC").
		Find(&conversations).Error
	if err != nil {
		return nil, err
	}

	return conversations, nil
}

func (r *conversationRepo) ListActiveMembers(ctx context.Context, conversationID uint64) ([]model.ConversationMember, error) {
	if conversationID == 0 {
		return nil, apperr.RequiredOne("conversation_id")
	}

	var members []model.ConversationMember
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND status = ?", conversationID, model.ConversationMemberStatusActive).
		Order("user_id ASC").
		Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

func (r *conversationRepo) CountActiveMembers(ctx context.Context, conversationID uint64) (int64, error) {
	if conversationID == 0 {
		return 0, apperr.RequiredOne("conversation_id")
	}
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND status = ?", conversationID, model.ConversationMemberStatusActive).
		Count(&count).Error
	return count, err
}

func (r *conversationRepo) GetMember(ctx context.Context, conversationID, userID uint64) (model.ConversationMember, error) {
	var member model.ConversationMember
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&member).Error
	return member, err
}

func (r *conversationRepo) UpsertMember(ctx context.Context, member *model.ConversationMember) error {
	if member == nil {
		return apperr.RequiredOne("member")
	}
	if member.ConversationID == 0 || member.UserID == 0 {
		return apperr.Required("conversation_id", "user_id")
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "conversation_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"role",
				"status",
				"visible",
				"invited_by",
				"joined_msg_seq",
				"updated_at",
			}),
		}).
		Create(member).Error
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

func (r *conversationRepo) UpdateName(ctx context.Context, conversationID uint64, name string) error {
	if conversationID == 0 {
		return apperr.RequiredOne("conversation_id")
	}
	result := r.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Update("name", name)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperr.NotFound(apperr.CodeConversationNotFound, "conversation not found")
	}
	return nil
}

func (r *conversationRepo) UpdateStatus(ctx context.Context, conversationID uint64, status model.ConversationStatus) error {
	if conversationID == 0 {
		return apperr.RequiredOne("conversation_id")
	}
	result := r.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperr.NotFound(apperr.CodeConversationNotFound, "conversation not found")
	}
	return nil
}

func (r *conversationRepo) UpdateMemberStatus(ctx context.Context, conversationID, userID uint64, status model.ConversationMemberStatus, visible bool) error {
	if conversationID == 0 || userID == 0 {
		return apperr.Required("conversation_id", "user_id")
	}
	result := r.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]any{
			"status":  status,
			"visible": visible,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperr.ConversationMemberNotFound()
	}
	return nil
}

func (r *conversationRepo) UpdateAllMemberStatus(ctx context.Context, conversationID uint64, status model.ConversationMemberStatus, visible bool) error {
	if conversationID == 0 {
		return apperr.RequiredOne("conversation_id")
	}
	return r.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ?", conversationID).
		Updates(map[string]any{
			"status":  status,
			"visible": visible,
		}).Error
}

func (r *conversationRepo) UpdateLastAckedMsgSeq(ctx context.Context, conversationID, userID, msgSeq uint64) error {
	if conversationID == 0 || userID == 0 || msgSeq == 0 {
		return apperr.Required("conversation_id", "user_id", "msg_seq")
	}

	result := r.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ? AND last_acked_msg_seq < ?", conversationID, userID, msgSeq).
		Update("last_acked_msg_seq", msgSeq)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		member, err := r.GetMember(ctx, conversationID, userID)
		if err != nil {
			return apperr.ConversationMemberNotFound()
		}
		if member.LastAckedMsgSeq >= msgSeq {
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
		Status:         model.ConversationMemberStatusActive,
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
