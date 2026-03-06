package repository

import (
	"context"

	"gorm.io/gorm"
	"github.com/zyy125/im-system/internal/model"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	return gorm.G[model.User](r.db).Create(ctx, user)
} 

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (model.User, error) {
	return gorm.G[model.User](r.db).Where("username = ?", username).First(ctx)
}