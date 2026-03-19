package repository

import (
	"context"

	"github.com/zyy125/im-system/internal/model"
	"gorm.io/gorm"
)

type MessageRepo interface {
	Create(ctx context.Context, msg *model.ChatMsg) error
}

type messageRepo struct {
	db *gorm.DB
}

func NewMessageRepo(db *gorm.DB) *messageRepo {
	return &messageRepo{db: db}
}

func (r *messageRepo) Create(ctx context.Context, msg *model.ChatMsg) error {
	return gorm.G[model.ChatMsg](r.db).Create(ctx, msg)
}
