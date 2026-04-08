package repository

import (
	"context"

	"github.com/zyy125/im-system/internal/model"
	"gorm.io/gorm"
)

type UserRepo interface {
	Create(ctx context.Context, user *model.User) error
	GetByUsername(ctx context.Context, username string) (model.User, error)
	GetByID(ctx context.Context, id uint64) (model.User, error)
	ListByIDs(ctx context.Context, ids []uint64) ([]model.User, error)
}

type userRepo struct {
	db *gorm.DB
}

var _ UserRepo = (*userRepo)(nil)

func NewUserRepo(db *gorm.DB) *userRepo {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *model.User) error {
	return gorm.G[model.User](r.db).Create(ctx, user)
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (model.User, error) {
	return gorm.G[model.User](r.db).Where("username = ?", username).First(ctx)
}

func (r *userRepo) GetByID(ctx context.Context, id uint64) (model.User, error) {
	return gorm.G[model.User](r.db).Where("id = ?", id).First(ctx)
}

func (r *userRepo) ListByIDs(ctx context.Context, ids []uint64) ([]model.User, error) {
	if len(ids) == 0 {
		return []model.User{}, nil
	}
	return gorm.G[model.User](r.db).Where("id IN ?", ids).Find(ctx)
}
