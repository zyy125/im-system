package repository

import (
	"log"

	"github.com/zyy125/im-system/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewMysql(dsn string) (*gorm.DB, error) {
	var err error

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		return nil, err
	}

	if err = AutoMigrate(db); err != nil {
		return nil, err
	}
	log.Println("MySQL connection and auto migration successful")

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&model.User{}, &model.ChatMsg{})
}
