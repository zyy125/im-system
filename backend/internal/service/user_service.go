package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/pkg/utils"
	"gorm.io/gorm"
)

type userService struct {
	userRepo     repository.UserRepo
	presenceRepo repository.PresenceRepo
	avatarDir    string
	avatarBase   string
}

type UserService interface {
	IsOnline(ctx context.Context, userID uint64) (bool, error)
	GetUserByID(ctx context.Context, userID uint64) (model.User, bool, error)
	UpdateAvatar(ctx context.Context, userID uint64, fileHeader *multipart.FileHeader) (string, error)
	ClearAvatar(ctx context.Context, userID uint64) error
}

var _ UserService = (*userService)(nil)

func NewUserService(userRepo repository.UserRepo, presenceRepo repository.PresenceRepo) UserService {
	return &userService{
		userRepo:     userRepo,
		presenceRepo: presenceRepo,
		avatarDir:    "./storage/avatars",
		avatarBase:   "/uploads/avatars",
	}
}

func (s *userService) IsOnline(ctx context.Context, userID uint64) (bool, error) {
	if userID == 0 {
		return false, apperr.RequiredOne("user_id")
	}
	return s.presenceRepo.IsOnline(ctx, userID)
}

func (s *userService) GetUserByID(ctx context.Context, userID uint64) (model.User, bool, error) {
	if userID == 0 {
		return model.User{}, false, apperr.RequiredOne("user_id")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, false, apperr.UserNotFound()
		}
		return model.User{}, false, err
	}
	online, err := s.presenceRepo.IsOnline(ctx, user.ID)
	if err != nil {
		return model.User{}, false, err
	}
	return user, online, nil
}

func (s *userService) UpdateAvatar(ctx context.Context, userID uint64, fileHeader *multipart.FileHeader) (string, error) {
	if userID == 0 {
		return "", apperr.RequiredOne("user_id")
	}
	if fileHeader == nil {
		return "", apperr.RequiredOne("file")
	}
	if fileHeader.Size <= 0 {
		return "", apperr.InvalidArgument("file is empty")
	}
	if fileHeader.Size > 2*1024*1024 {
		return "", apperr.InvalidArgument("avatar file is too large")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		return "", apperr.InvalidArgument("unsupported avatar file type")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	if err := os.MkdirAll(s.avatarDir, 0o755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s%s", utils.GenerateUUID(), ext)
	dstPath := filepath.Join(s.avatarDir, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	oldAvatarPath := user.Avatar
	user.Avatar = fmt.Sprintf("%s/%s", strings.TrimRight(s.avatarBase, "/"), filename)
	if err := s.userRepo.Update(ctx, &user); err != nil {
		return "", err
	}

	if oldAvatarPath != "" && strings.HasPrefix(oldAvatarPath, strings.TrimRight(s.avatarBase, "/")+"/") {
		oldFilename := strings.TrimPrefix(oldAvatarPath, strings.TrimRight(s.avatarBase, "/")+"/")
		_ = os.Remove(filepath.Join(s.avatarDir, oldFilename))
	}

	return user.Avatar, nil
}

func (s *userService) ClearAvatar(ctx context.Context, userID uint64) error {
	if userID == 0 {
		return apperr.RequiredOne("user_id")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	oldAvatarPath := user.Avatar
	user.Avatar = ""
	if err := s.userRepo.Update(ctx, &user); err != nil {
		return err
	}

	if oldAvatarPath != "" && strings.HasPrefix(oldAvatarPath, strings.TrimRight(s.avatarBase, "/")+"/") {
		oldFilename := strings.TrimPrefix(oldAvatarPath, strings.TrimRight(s.avatarBase, "/")+"/")
		_ = os.Remove(filepath.Join(s.avatarDir, oldFilename))
	}
	return nil
}
