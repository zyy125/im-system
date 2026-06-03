package infra

import (
	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewMySQL(cfg config.Mysql) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := AutoMigrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.User{},
		&model.Message{},
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
