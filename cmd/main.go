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
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return app.Run(ctx, cfg)
}
