package app

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/handler"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/internal/router"
	"github.com/zyy125/im-system/internal/service"
)

type App struct {
	Router *gin.Engine
}

func InitApp(cfg *config.Config) (*App, error) {
	db, err := repository.InitDB(cfg.Mysql.DSN)
	if err != nil {
		return nil, err
	}
	rdb, err := repository.InitRedisClient(context.Background(), cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return nil, err
	}

	userRepo := repository.NewUserRepo(db)
	blacklistRepo := repository.NewTokenBlacklistRepo(rdb)

	userSvc := service.NewUserService(userRepo, &cfg.JWT, blacklistRepo)

	userHandler := handler.NewUserHandler(userSvc)

	initRouterParams := &router.InitRouterParams{
		UserHandler: userHandler,
		BlacklistRepo: blacklistRepo,
		JwtCfg: &cfg.JWT,
	}
	router := router.InitRouter(initRouterParams)

	return &App{Router: router}, nil
}