package service

import (
	"context"
	"errors"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
	"gorm.io/gorm"
)

type FriendRequestSummary struct {
	ID        uint64
	Status    model.FriendRequestStatus
	Message   string
	Requester FriendRequestUser
	Receiver  FriendRequestUser
}

type FriendRequestUser struct {
	ID       uint64
	Username string
	Online   bool
}

type friendRequestService struct {
	friendRequestRepo repository.FriendRequestRepo
	friendService     FriendService
	userRepo          repository.UserRepo
	presenceRepo      repository.PresenceRepo
}

type FriendRequestService interface {
	Send(ctx context.Context, requesterID, receiverID uint64, message string) (string, error)
	Accept(ctx context.Context, userID, requestID uint64) error
	Reject(ctx context.Context, userID, requestID uint64) error
	ListIncoming(ctx context.Context, userID uint64) ([]FriendRequestSummary, error)
	ListOutgoing(ctx context.Context, userID uint64) ([]FriendRequestSummary, error)
}

var _ FriendRequestService = (*friendRequestService)(nil)

func NewFriendRequestService(
	friendRequestRepo repository.FriendRequestRepo,
	friendService FriendService,
	userRepo repository.UserRepo,
	presenceRepo repository.PresenceRepo,
) FriendRequestService {
	return &friendRequestService{
		friendRequestRepo: friendRequestRepo,
		friendService:     friendService,
		userRepo:          userRepo,
		presenceRepo:      presenceRepo,
	}
}

func (s *friendRequestService) Send(ctx context.Context, requesterID, receiverID uint64, message string) (string, error) {
	if requesterID == 0 || receiverID == 0 {
		return "", apperr.Required("requester_id", "receiver_id")
	}
	if requesterID == receiverID {
		return "", apperr.FriendCannotAddSelf()
	}
	if _, err := s.userRepo.GetByID(ctx, receiverID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", apperr.UserNotFound()
		}
		return "", err
	}

	areFriends, err := s.friendService.AreFriends(ctx, requesterID, receiverID)
	if err != nil {
		return "", err
	}
	if areFriends {
		return "already_friends", nil
	}

	// Check if there's a pending request in the opposite direction to auto-accept
	_, err = s.friendRequestRepo.FindPendingBetween(ctx, receiverID, requesterID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	if err == nil {
		if err := s.friendService.AddFriend(ctx, requesterID, receiverID); err != nil {
			return "", err
		}
		if err := s.friendRequestRepo.ResolvePendingBetween(ctx, requesterID, receiverID, model.FriendRequestAccepted); err != nil {
			return "", err
		}
		return "auto_accepted", nil
	}

	// Check if there's already a pending request in the same direction
	_, err = s.friendRequestRepo.FindPendingBetween(ctx, requesterID, receiverID)
	if err == nil {
		return "pending", nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	req := &model.FriendRequest{
		RequesterID: requesterID,
		ReceiverID:  receiverID,
		Status:      model.FriendRequestPending,
		Message:     message,
	}
	if err := s.friendRequestRepo.Create(ctx, req); err != nil {
		return "", err
	}
	return "pending", nil
}

func (s *friendRequestService) Accept(ctx context.Context, userID, requestID uint64) error {
	req, err := s.friendRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.FriendRequestNotFound()
		}
		return err
	}
	if req.ReceiverID != userID {
		return apperr.FriendRequestNoPermission("accept")
	}
	if !req.IsPending() {
		return apperr.FriendRequestNotPending()
	}

	if err := s.friendService.AddFriend(ctx, req.RequesterID, req.ReceiverID); err != nil {
		return err
	}
	return s.friendRequestRepo.ResolvePendingBetween(ctx, req.RequesterID, req.ReceiverID, model.FriendRequestAccepted)
}

func (s *friendRequestService) Reject(ctx context.Context, userID, requestID uint64) error {
	req, err := s.friendRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.FriendRequestNotFound()
		}
		return err
	}
	if req.ReceiverID != userID {
		return apperr.FriendRequestNoPermission("reject")
	}
	if !req.IsPending() {
		return apperr.FriendRequestNotPending()
	}
	return s.friendRequestRepo.UpdateStatus(ctx, req.ID, model.FriendRequestRejected)
}

func (s *friendRequestService) ListIncoming(ctx context.Context, userID uint64) ([]FriendRequestSummary, error) {
	if userID == 0 {
		return nil, apperr.RequiredOne("user_id")
	}

	reqs, err := s.friendRequestRepo.ListIncomingPending(ctx, userID)
	if err != nil {
		return nil, err
	}
	reqs, err = s.filterActiveRequests(ctx, reqs)
	if err != nil {
		return nil, err
	}
	return s.buildSummaries(ctx, reqs)
}

func (s *friendRequestService) ListOutgoing(ctx context.Context, userID uint64) ([]FriendRequestSummary, error) {
	if userID == 0 {
		return nil, apperr.RequiredOne("user_id")
	}

	reqs, err := s.friendRequestRepo.ListOutgoingPending(ctx, userID)
	if err != nil {
		return nil, err
	}
	reqs, err = s.filterActiveRequests(ctx, reqs)
	if err != nil {
		return nil, err
	}
	return s.buildSummaries(ctx, reqs)
}

func (s *friendRequestService) buildSummaries(ctx context.Context, reqs []model.FriendRequest) ([]FriendRequestSummary, error) {
	res := make([]FriendRequestSummary, 0, len(reqs))
	for _, req := range reqs {
		requester, err := s.buildUser(ctx, req.RequesterID)
		if err != nil {
			return nil, err
		}
		receiver, err := s.buildUser(ctx, req.ReceiverID)
		if err != nil {
			return nil, err
		}
		res = append(res, FriendRequestSummary{
			ID:        req.ID,
			Status:    req.Status,
			Message:   req.Message,
			Requester: requester,
			Receiver:  receiver,
		})
	}
	return res, nil
}

func (s *friendRequestService) buildUser(ctx context.Context, userID uint64) (FriendRequestUser, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return FriendRequestUser{}, err
	}
	online, err := s.presenceRepo.IsOnline(ctx, userID)
	if err != nil {
		return FriendRequestUser{}, err
	}
	return FriendRequestUser{
		ID:       userID,
		Username: user.Username,
		Online:   online,
	}, nil
}

func (s *friendRequestService) filterActiveRequests(ctx context.Context, reqs []model.FriendRequest) ([]model.FriendRequest, error) {
	filtered := make([]model.FriendRequest, 0, len(reqs))
	for _, req := range reqs {
		areFriends, err := s.friendService.AreFriends(ctx, req.RequesterID, req.ReceiverID)
		if err != nil {
			return nil, err
		}
		if areFriends {
			continue
		}
		filtered = append(filtered, req)
	}
	return filtered, nil
}
