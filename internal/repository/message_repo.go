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
	// Create 插入一条消息；若 msg_id 已存在则回填已有记录实现幂等。
	Create(ctx context.Context, msg *model.Message) error
	// ListConversationHistory 按会话分页读取历史消息。
	ListConversationHistory(ctx context.Context, conversationID uint64, limit int, beforeSeq, afterSeq uint64) ([]model.Message, bool, error)
	// ListConversationAfterSeq 按 seq 正序读取某条 seq 之后的消息，用于补洞和同步。
	ListConversationAfterSeq(ctx context.Context, conversationID, afterSeq uint64, limit int) ([]model.Message, bool, error)
	// ListConversationRangeAfterSeq 按 seq 正序读取 (afterSeq, untilSeq] 区间内的消息。
	ListConversationRangeAfterSeq(ctx context.Context, conversationID, afterSeq, untilSeq uint64, limit int) ([]model.Message, bool, error)
	// GetLatestByConversation 获取某个会话最新的一条消息。
	GetLatestByConversation(ctx context.Context, conversationID uint64) (model.Message, error)
	// GetMaxSeqByConversation 查询某个会话当前已持久化的最大 seq。
	GetMaxSeqByConversation(ctx context.Context, conversationID uint64) (uint64, error)
	// ListConversationIDs 返回当前存在消息的会话 ID 列表。
	ListConversationIDs(ctx context.Context) ([]uint64, error)
	// CountUnreadByConversation 统计某个成员在会话中的未读消息数。
	CountUnreadByConversation(ctx context.Context, conversationID, userID, afterSeq uint64) (int64, error)
}

type messageRepo struct {
	db *gorm.DB
}

func NewMessageRepo(db *gorm.DB) *messageRepo {
	return &messageRepo{db: db}
}

func (r *messageRepo) Create(ctx context.Context, msg *model.Message) error {
	if msg.MsgID == "" {
		return apperr.MessageIDRequired()
	}
	if msg.ConversationID == 0 || msg.Seq == 0 {
		return apperr.Required("conversation_id", "seq")
	}

	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(msg)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		existing, err := r.getByMsgID(ctx, msg.MsgID)
		if err == nil {
			*msg = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		existing, err = r.getByConversationAndSeq(ctx, msg.ConversationID, msg.Seq)
		if err != nil {
			return err
		}
		*msg = existing
	}
	return nil
}

func (r *messageRepo) ListConversationHistory(ctx context.Context, conversationID uint64, limit int, beforeSeq, afterSeq uint64) ([]model.Message, bool, error) {
	if conversationID == 0 {
		return nil, false, apperr.RequiredOne("conversation_id")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	q := r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("conversation_id = ?", conversationID)
	if beforeSeq > 0 {
		q = q.Where("seq < ?", beforeSeq)
	}
	if afterSeq > 0 {
		q = q.Where("seq > ?", afterSeq)
	}

	var msgs []model.Message
	if err := q.Order("seq DESC").Limit(limit + 1).Find(&msgs).Error; err != nil {
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

func (r *messageRepo) ListConversationAfterSeq(ctx context.Context, conversationID, afterSeq uint64, limit int) ([]model.Message, bool, error) {
	if conversationID == 0 {
		return nil, false, apperr.RequiredOne("conversation_id")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}

	var msgs []model.Message
	if err := r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("conversation_id = ? AND seq > ?", conversationID, afterSeq).
		Order("seq ASC").
		Limit(limit + 1).
		Find(&msgs).Error; err != nil {
		return nil, false, err
	}

	hasMore := len(msgs) > limit
	if hasMore {
		msgs = msgs[:limit]
	}
	return msgs, hasMore, nil
}

func (r *messageRepo) ListConversationRangeAfterSeq(ctx context.Context, conversationID, afterSeq, untilSeq uint64, limit int) ([]model.Message, bool, error) {
	if conversationID == 0 {
		return nil, false, apperr.RequiredOne("conversation_id")
	}
	if untilSeq == 0 {
		return []model.Message{}, false, nil
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}

	var msgs []model.Message
	if err := r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("conversation_id = ? AND seq > ? AND seq <= ?", conversationID, afterSeq, untilSeq).
		Order("seq ASC").
		Limit(limit + 1).
		Find(&msgs).Error; err != nil {
		return nil, false, err
	}

	hasMore := len(msgs) > limit
	if hasMore {
		msgs = msgs[:limit]
	}
	return msgs, hasMore, nil
}

func (r *messageRepo) GetLatestByConversation(ctx context.Context, conversationID uint64) (model.Message, error) {
	if conversationID == 0 {
		return model.Message{}, apperr.MessageConversationRequired()
	}

	var msg model.Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("seq DESC").
		First(&msg).Error
	return msg, err
}

func (r *messageRepo) GetMaxSeqByConversation(ctx context.Context, conversationID uint64) (uint64, error) {
	if conversationID == 0 {
		return 0, apperr.RequiredOne("conversation_id")
	}
	type result struct {
		MaxSeq uint64
	}
	var row result
	err := r.db.WithContext(ctx).
		Model(&model.Message{}).
		Select("COALESCE(MAX(seq), 0) AS max_seq").
		Where("conversation_id = ?", conversationID).
		Scan(&row).Error
	return row.MaxSeq, err
}

func (r *messageRepo) ListConversationIDs(ctx context.Context) ([]uint64, error) {
	var ids []uint64
	err := r.db.WithContext(ctx).
		Model(&model.Message{}).
		Distinct("conversation_id").
		Order("conversation_id ASC").
		Pluck("conversation_id", &ids).Error
	return ids, err
}

func (r *messageRepo) CountUnreadByConversation(ctx context.Context, conversationID, userID, afterSeq uint64) (int64, error) {
	if conversationID == 0 || userID == 0 {
		return 0, apperr.Required("conversation_id", "user_id")
	}

	q := r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("conversation_id = ? AND `from` <> ?", conversationID, userID)

	if afterSeq > 0 {
		q = q.Where("seq > ?", afterSeq)
	}

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *messageRepo) getByMsgID(ctx context.Context, msgID string) (model.Message, error) {
	var msg model.Message
	err := r.db.WithContext(ctx).
		Where("msg_id = ?", msgID).
		First(&msg).Error
	return msg, err
}

func (r *messageRepo) getByConversationAndSeq(ctx context.Context, conversationID, seq uint64) (model.Message, error) {
	var msg model.Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND seq = ?", conversationID, seq).
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
