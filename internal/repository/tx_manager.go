package repository

import (
	"context"

	"gorm.io/gorm"
)

type MessageTxManager interface {
	WithinMessageTx(ctx context.Context, fn func(messageRepo MessageRepo, conversationRepo ConversationRepo) error) error
}

type gormMessageTxManager struct {
	db *gorm.DB
}

var _ MessageTxManager = (*gormMessageTxManager)(nil)

func NewMessageTxManager(db *gorm.DB) MessageTxManager {
	return &gormMessageTxManager{db: db}
}

func (m *gormMessageTxManager) WithinMessageTx(ctx context.Context, fn func(messageRepo MessageRepo, conversationRepo ConversationRepo) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(NewMessageRepo(tx), NewConversationRepo(tx))
	})
}
