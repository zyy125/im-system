package service

import (
	"context"
	"errors"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/repository"
	"gorm.io/gorm"
)

type friendService struct {
	friendRepo       repository.FriendRepo
	userRepo         repository.UserRepo
	presenceRepo     repository.PresenceRepo
	conversationRepo repository.ConversationRepo
}

type FriendService interface {
	AddFriend(ctx context.Context, userID, friendID uint64) error
	RemoveFriend(ctx context.Context, userID, friendID uint64) error
	AreFriends(ctx context.Context, userID, friendID uint64) (bool, error)
	ListFriends(ctx context.Context, userID uint64) ([]FriendInfo, error)
}

var _ FriendService = (*friendService)(nil)

func NewFriendService(
	friendRepo repository.FriendRepo,
	userRepo repository.UserRepo,
	presenceRepo repository.PresenceRepo,
	conversationRepo repository.ConversationRepo,
) FriendService {
	return &friendService{
		friendRepo:       friendRepo,
		userRepo:         userRepo,
		presenceRepo:     presenceRepo,
		conversationRepo: conversationRepo,
	}
}

func (s *friendService) AddFriend(ctx context.Context, userID, friendID uint64) error {
	if userID == 0 || friendID == 0 {
		return apperr.Required("user_id", "friend_id")
	}
	if userID == friendID {
		return apperr.FriendCannotAddSelf()
	}
	if _, err := s.userRepo.GetByID(ctx, friendID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.UserNotFound()
		}
		return err
	}

	alreadyFriends, err := s.friendRepo.AreFriends(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if !alreadyFriends {
		if err := s.friendRepo.AddPair(ctx, userID, friendID); err != nil {
			return err
		}
	}
	conversation, err := s.conversationRepo.GetOrCreateSingle(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if err := s.conversationRepo.SetVisible(ctx, conversation.ID, userID, true); err != nil {
		return err
	}
	return s.conversationRepo.SetVisible(ctx, conversation.ID, friendID, true)
}

func (s *friendService) RemoveFriend(ctx context.Context, userID, friendID uint64) error {
	if userID == 0 || friendID == 0 {
		return apperr.Required("user_id", "friend_id")
	}
	return s.friendRepo.RemovePair(ctx, userID, friendID)
}

func (s *friendService) AreFriends(ctx context.Context, userID, friendID uint64) (bool, error) {
	if userID == 0 || friendID == 0 {
		return false, apperr.Required("user_id", "friend_id")
	}
	return s.friendRepo.AreFriends(ctx, userID, friendID)
}

type FriendInfo struct {
	UserID   uint64
	Username string
	Online   bool
}

func (s *friendService) ListFriends(ctx context.Context, userID uint64) ([]FriendInfo, error) {
	if userID == 0 {
		return nil, apperr.RequiredOne("user_id")
	}

	ids, err := s.friendRepo.ListFriendIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []FriendInfo{}, nil
	}
	users, err := s.userRepo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make([]FriendInfo, 0, len(users))
	for _, u := range users {
		online, err := s.presenceRepo.IsOnline(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		res = append(res, FriendInfo{UserID: u.ID, Username: u.Username, Online: online})
	}
	return res, nil
}
