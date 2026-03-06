package repository

import (
	"gorm.io/gorm"
	"gorm.io/driver/mysql"
	"log"
	"github.com/zyy125/im-system/internal/model"
)

var DB *gorm.DB

func InitDB(dsn string) {
	var err error

	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalf("MySQL connection failed: %v", err)
	}

	if err = DB.AutoMigrate(model.User{}); err != nil {
		log.Fatalf("MySQL auto migration failed: %v", err)
	}

	log.Println("MySQL connection and auto migration successful")
}