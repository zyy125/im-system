package repository

import (
	"context"
	"errors"

	"github.com/go-sql-driver/mysql"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MessageRepo interface {
	Create(ctx context.Context, msg *model.ChatMessage) error
	ListBetween(ctx context.Context, userID, peerID uint64, limit int, beforeID uint64) ([]model.ChatMessage, bool, error)
	// ListConversationPending returns all messages in a conversation whose ID is in
	// the range (afterSeq, untilSeq]. It is conversation-scoped and does not filter
	// by receiver, so both outbound and inbound messages may be returned.
	ListConversationPending(ctx context.Context, conversationID string, afterSeq, untilSeq uint64) ([]model.ChatMessage, error)
	// ListConversationPendingForUser returns only messages addressed to userID in a
	// conversation whose ID is in the range (afterSeq, untilSeq]. Use this when you
	// need the current user's inbound pending messages, such as unread/offline sync.
	ListConversationPendingForUser(ctx context.Context, conversationID string, userID, afterSeq, untilSeq uint64) ([]model.ChatMessage, error)
	GetLatestByConversation(ctx context.Context, conversationID string) (model.ChatMessage, error)
	CountUnreadByConversation(ctx context.Context, conversationID string, userID uint64, afterSeq uint64) (int64, error)
	GetByConversationAndMsgID(ctx context.Context, conversationID, msgID string) (model.ChatMessage, error)
}

type messageRepo struct {
	db *gorm.DB
}

func NewMessageRepo(db *gorm.DB) *messageRepo {
	return &messageRepo{db: db}
}

func (r *messageRepo) Create(ctx context.Context, msg *model.ChatMessage) error {
	if msg.MsgID == "" {
		return apperr.MessageIDRequired()
	}

	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(msg)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		existing, err := r.getByMsgID(ctx, msg.MsgID)
		if err != nil {
			return err
		}
		*msg = existing
	}
	return nil
}

func (r *messageRepo) ListBetween(ctx context.Context, userID, peerID uint64, limit int, beforeID uint64) ([]model.ChatMessage, bool, error) {
	if userID == 0 || peerID == 0 {
		return nil, false, apperr.Required("user_id", "peer_id")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	q := r.db.WithContext(ctx).
		Model(&model.ChatMessage{}).
		Where("( `from` = ? AND `to` = ? ) OR ( `from` = ? AND `to` = ? )", userID, peerID, peerID, userID)
	if beforeID > 0 {
		q = q.Where("id < ?", beforeID)
	}

	var msgs []model.ChatMessage
	if err := q.Order("id DESC").Limit(limit + 1).Find(&msgs).Error; err != nil {
		return nil, false, err
	}

	hasMore := len(msgs) > limit
	if hasMore {
		msgs = msgs[:limit]
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return msgs, hasMore, nil
}

func (r *messageRepo) ListConversationPending(ctx context.Context, conversationID string, afterSeq, untilSeq uint64) ([]model.ChatMessage, error) {
	if conversationID == "" {
		return nil, apperr.MessageConversationRequired()
	}
	if untilSeq == 0 {
		return []model.ChatMessage{}, nil
	}

	q := r.db.WithContext(ctx).
		Model(&model.ChatMessage{}).
		Where("conversation_id = ?", conversationID)

	if afterSeq > 0 {
		q = q.Where("id > ?", afterSeq)
	}

	var msgs []model.ChatMessage
	err := q.Where("id <= ?", untilSeq).
		Order("id ASC").
		Find(&msgs).Error
	if err != nil {
		return nil, err
	}

	return msgs, nil
}

func (r *messageRepo) ListConversationPendingForUser(ctx context.Context, conversationID string, userID, afterSeq, untilSeq uint64) ([]model.ChatMessage, error) {
	if conversationID == "" || userID == 0 {
		return nil, apperr.Required("conversation_id", "user_id")
	}
	if untilSeq == 0 {
		return []model.ChatMessage{}, nil
	}

	q := r.db.WithContext(ctx).
		Model(&model.ChatMessage{}).
		Where("conversation_id = ? AND `to` = ?", conversationID, userID)

	if afterSeq > 0 {
		q = q.Where("id > ?", afterSeq)
	}

	var msgs []model.ChatMessage
	err := q.Where("id <= ?", untilSeq).
		Order("id ASC").
		Find(&msgs).Error
	if err != nil {
		return nil, err
	}

	return msgs, nil
}

func (r *messageRepo) GetLatestByConversation(ctx context.Context, conversationID string) (model.ChatMessage, error) {
	if conversationID == "" {
		return model.ChatMessage{}, apperr.MessageConversationRequired()
	}

	var msg model.ChatMessage
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("id DESC").
		First(&msg).Error
	return msg, err
}

func (r *messageRepo) CountUnreadByConversation(ctx context.Context, conversationID string, userID uint64, afterSeq uint64) (int64, error) {
	if conversationID == "" || userID == 0 {
		return 0, apperr.Required("conversation_id", "user_id")
	}

	q := r.db.WithContext(ctx).
		Model(&model.ChatMessage{}).
		Where("conversation_id = ? AND `to` = ?", conversationID, userID)

	if afterSeq > 0 {
		q = q.Where("id > ?", afterSeq)
	}

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *messageRepo) GetByConversationAndMsgID(ctx context.Context, conversationID, msgID string) (model.ChatMessage, error) {
	if conversationID == "" || msgID == "" {
		return model.ChatMessage{}, apperr.Required("conversation_id", "msg_id")
	}
	var msg model.ChatMessage
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND msg_id = ?", conversationID, msgID).
		First(&msg).Error
	return msg, err
}

func (r *messageRepo) getByMsgID(ctx context.Context, msgID string) (model.ChatMessage, error) {
	var msg model.ChatMessage
	err := r.db.WithContext(ctx).
		Where("msg_id = ?", msgID).
		First(&msg).Error
	return msg, err
}

func isDuplicateKeyErr(err error) bool {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1062
	}
	return false
}
