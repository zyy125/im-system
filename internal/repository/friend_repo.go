package repository

import (
	"context"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FriendRepo interface {
	AddPair(ctx context.Context, userID, friendID uint64) error
	RemovePair(ctx context.Context, userID, friendID uint64) error
	AreFriends(ctx context.Context, userID, friendID uint64) (bool, error)
	ListFriendIDs(ctx context.Context, userID uint64) ([]uint64, error)
}

type friendRepo struct {
	db *gorm.DB
}

var _ FriendRepo = (*friendRepo)(nil)

func NewFriendRepo(db *gorm.DB) *friendRepo {
	return &friendRepo{db: db}
}

func (r *friendRepo) AddPair(ctx context.Context, userID, friendID uint64) error {
	if userID == 0 || friendID == 0 {
		return apperr.Required("user_id", "friend_id")
	}
	if userID == friendID {
		return apperr.FriendCannotAddSelf()
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&model.Friend{UserID: userID, FriendID: friendID}).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&model.Friend{UserID: friendID, FriendID: userID}).Error; err != nil {
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
