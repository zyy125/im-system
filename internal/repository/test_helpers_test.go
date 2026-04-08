package repository

import (
	"testing"

	"github.com/zyy125/im-system/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.ChatMessage{},
		&model.Friend{},
		&model.FriendRequest{},
		&model.Conversation{},
		&model.ConversationMember{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	return db
}
