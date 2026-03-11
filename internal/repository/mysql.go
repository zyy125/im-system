package repository

import (
	"gorm.io/gorm"
	"gorm.io/driver/mysql"
	"log"
	"github.com/zyy125/im-system/internal/model"
)

func InitDB(dsn string) (*gorm.DB, error) {
	var err error

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalf("MySQL connection failed: %v", err)
	}

	if err = AutoMigrate(db); err != nil {
		log.Fatalf("MySQL auto migration failed: %v", err)
	}
	log.Println("MySQL connection and auto migration successful")

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&model.User{})
}



