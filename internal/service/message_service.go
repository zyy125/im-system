package service

import (
	"context"
	"errors"
	"time"

	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
)

type MessageService struct {
	msgRepo repository.MessageRepo
}

func NewMessageService(msgRepo repository.MessageRepo) *MessageService {
	return &MessageService{msgRepo: msgRepo}
}

func (s *MessageService) SaveMsg(ctx context.Context, msg *model.ChatMsg) error {
	if msg.MsgID == "" {
		return errors.New("msg_id is empty")
	}
	now := time.Now().UnixMilli()

	msg.SendTime = now

	return s.msgRepo.Create(ctx, msg)
}
