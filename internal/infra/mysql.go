package infra

import (
	"log"

	"github.com/zyy125/im-system/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewMySQL(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := AutoMigrate(db); err != nil {
		return nil, err
	}

	log.Println("MySQL connection and auto migration successful")
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.User{},
		&model.ChatMessage{},
		&model.Friend{},
		&model.FriendRequest{},
		&model.Conversation{},
		&model.ConversationMember{},
	); err != nil {
		return err
	}

	if err := normalizeConversationSingleKey(db); err != nil {
		return err
	}
	return nil
}

func normalizeConversationSingleKey(db *gorm.DB) error {
	return db.Model(&model.Conversation{}).
		Where("type = ? AND single_key = ''", model.ConversationTypeGroup).
		Update("single_key", nil).Error
}
