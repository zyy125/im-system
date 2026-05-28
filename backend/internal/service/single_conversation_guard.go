package service

import (
	"context"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
)

func ensureSingleConversationFriendship(ctx context.Context, friendRepo repository.FriendRepo, conv model.Conversation, userID uint64) error {
	if !conv.IsSingle() {
		return nil
	}
	if friendRepo == nil {
		return nil
	}

	peerID, err := extractPeerID(conv, userID)
	if err != nil {
		return err
	}
	areFriends, err := friendRepo.AreFriends(ctx, userID, peerID)
	if err != nil {
		return err
	}
	if !areFriends {
		return apperr.ConversationNotAccessible()
	}
	return nil
}
