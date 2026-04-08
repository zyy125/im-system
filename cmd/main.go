// @title IM System API
// @version 1.0
// @description This is a IM System API.
// @host localhost:8080
// @BasePath /

// JWT认证
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

package main

import (
	"context"
	"log"

	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/app"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app, err := app.InitApp(cfg, ctx)
	if err != nil {
		log.Fatalf("Error initializing app: %v", err)
	}

	if err := app.Router.Run(":8080"); err != nil {
		log.Fatalf("Error running app: %v", err)
	}
}
