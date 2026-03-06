package main

import (
	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/repository"
)

func main() {
	dsn := "root:123456@tcp(127.0.0.1:3306)/im_db?charset=utf8mb4&parseTime=True&loc=Local"
	repository.InitDB(dsn)

	r := gin.Default()
	r.GET("/ping", func (c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.Run(":8080")
}