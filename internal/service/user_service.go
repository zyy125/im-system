package service

import (
	"context"

	"github.com/zyy125/im-system/internal/repository"
)



type UserService struct {
	userRepo repository.UserRepo
	presenceRepo repository.PresenceRepo
}

func NewUserService(userRepo repository.UserRepo, presenceRepo repository.PresenceRepo) *UserService {
	return &UserService{userRepo: userRepo, presenceRepo: presenceRepo}
}

func (s *UserService) CheckUserOnline(ctx context.Context, userID uint64) (bool, error) {
	return s.presenceRepo.IsOnline(ctx, userID)
}