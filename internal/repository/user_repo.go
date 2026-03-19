package repository

import (
	"context"

	"gorm.io/gorm"
	"github.com/zyy125/im-system/internal/model"
)

type UserRepo interface {
    Create(ctx context.Context, user *model.User) error
    GetByUsername(ctx context.Context, username string) (model.User, error)
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

