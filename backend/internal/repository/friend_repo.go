package repository

import (
	"context"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FriendRepo interface {
	AddPair(ctx context.Context, userID, friendID, conversationID uint64) error
	RemovePair(ctx context.Context, userID, friendID uint64) error
	AreFriends(ctx context.Context, userID, friendID uint64) (bool, error)
	ListFriendIDs(ctx context.Context, userID uint64) ([]uint64, error)
	ListFriendProfiles(ctx context.Context, userID uint64) ([]FriendProfile, error)
}

type FriendProfile struct {
	UserID         uint64
	Username       string
	ConversationID uint64
}

type friendRepo struct {
	db *gorm.DB
}

var _ FriendRepo = (*friendRepo)(nil)

func NewFriendRepo(db *gorm.DB) *friendRepo {
	return &friendRepo{db: db}
}

func (r *friendRepo) AddPair(ctx context.Context, userID, friendID, conversationID uint64) error {
	if userID == 0 || friendID == 0 || conversationID == 0 {
		return apperr.Required("user_id", "friend_id", "conversation_id")
	}
	if userID == friendID {
		return apperr.FriendCannotAddSelf()
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "friend_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"conversation_id"}),
		}).Create(&model.Friend{UserID: userID, FriendID: friendID, ConversationID: conversationID}).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "friend_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"conversation_id"}),
		}).Create(&model.Friend{UserID: friendID, FriendID: userID, ConversationID: conversationID}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *friendRepo) RemovePair(ctx context.Context, userID, friendID uint64) error {
	if userID == 0 || friendID == 0 {
		return apperr.Required("user_id", "friend_id")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND friend_id = ?", userID, friendID).Delete(&model.Friend{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND friend_id = ?", friendID, userID).Delete(&model.Friend{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *friendRepo) AreFriends(ctx context.Context, userID, friendID uint64) (bool, error) {
	if userID == 0 || friendID == 0 {
		return false, apperr.Required("user_id", "friend_id")
	}
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Friend{}).Where("user_id = ? AND friend_id = ?", userID, friendID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *friendRepo) ListFriendProfiles(ctx context.Context, userID uint64) ([]FriendProfile, error) {
	if userID == 0 {
		return nil, apperr.RequiredOne("user_id")
	}

	var items []FriendProfile
	err := r.db.WithContext(ctx).
		Table("friends").
		Select("friends.friend_id AS user_id, users.username, friends.conversation_id").
		Joins("JOIN users ON users.id = friends.friend_id").
		Where("friends.user_id = ?", userID).
		Order("friends.friend_id ASC").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *friendRepo) ListFriendIDs(ctx context.Context, userID uint64) ([]uint64, error) {
	if userID == 0 {
		return nil, apperr.RequiredOne("user_id")
	}
	var ids []uint64
	if err := r.db.WithContext(ctx).Model(&model.Friend{}).Select("friend_id").Where("user_id = ?", userID).Scan(&ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}
