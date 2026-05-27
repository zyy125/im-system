package repository

import (
	"context"
	"time"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"gorm.io/gorm"
)

type FriendRequestRepo interface {
	Create(ctx context.Context, req *model.FriendRequest) error
	GetByID(ctx context.Context, id uint64) (model.FriendRequest, error)
	FindPendingBetween(ctx context.Context, requesterID, receiverID uint64) (model.FriendRequest, error)
	ListIncomingPending(ctx context.Context, userID uint64) ([]model.FriendRequest, error)
	ListOutgoingPending(ctx context.Context, userID uint64) ([]model.FriendRequest, error)
	UpdateStatus(ctx context.Context, id uint64, status model.FriendRequestStatus) error
	ResolvePendingBetween(ctx context.Context, userA, userB uint64, status model.FriendRequestStatus) error
}

type friendRequestRepo struct {
	db *gorm.DB
}

var _ FriendRequestRepo = (*friendRequestRepo)(nil)

func NewFriendRequestRepo(db *gorm.DB) *friendRequestRepo {
	return &friendRequestRepo{db: db}
}

func (r *friendRequestRepo) Create(ctx context.Context, req *model.FriendRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *friendRequestRepo) GetByID(ctx context.Context, id uint64) (model.FriendRequest, error) {
	var req model.FriendRequest
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&req).Error
	return req, err
}

func (r *friendRequestRepo) FindPendingBetween(ctx context.Context, requesterID, receiverID uint64) (model.FriendRequest, error) {
	if requesterID == 0 || receiverID == 0 {
		return model.FriendRequest{}, apperr.Required("requester_id", "receiver_id")
	}

	var req model.FriendRequest
	err := r.db.WithContext(ctx).
		Where("requester_id = ? AND receiver_id = ? AND status = ?", requesterID, receiverID, model.FriendRequestPending).
		Order("id DESC").
		First(&req).Error
	return req, err
}

func (r *friendRequestRepo) ListIncomingPending(ctx context.Context, userID uint64) ([]model.FriendRequest, error) {
	var reqs []model.FriendRequest
	err := r.db.WithContext(ctx).
		Where("receiver_id = ? AND status = ?", userID, model.FriendRequestPending).
		Order("id DESC").
		Find(&reqs).Error
	return reqs, err
}

func (r *friendRequestRepo) ListOutgoingPending(ctx context.Context, userID uint64) ([]model.FriendRequest, error) {
	var reqs []model.FriendRequest
	err := r.db.WithContext(ctx).
		Where("requester_id = ? AND status = ?", userID, model.FriendRequestPending).
		Order("id DESC").
		Find(&reqs).Error
	return reqs, err
}

func (r *friendRequestRepo) UpdateStatus(ctx context.Context, id uint64, status model.FriendRequestStatus) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.FriendRequest{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     status,
			"handled_at": &now,
		}).Error
}

func (r *friendRequestRepo) ResolvePendingBetween(ctx context.Context, userA, userB uint64, status model.FriendRequestStatus) error {
	if userA == 0 || userB == 0 {
		return apperr.Required("user_a", "user_b")
	}

	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.FriendRequest{}).
		Where(
			"status = ? AND ((requester_id = ? AND receiver_id = ?) OR (requester_id = ? AND receiver_id = ?))",
			model.FriendRequestPending,
			userA, userB,
			userB, userA,
		).
		Updates(map[string]any{
			"status":     status,
			"handled_at": &now,
		}).Error
}
