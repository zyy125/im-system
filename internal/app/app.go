package app

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/handler"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/internal/router"
	"github.com/zyy125/im-system/internal/service"
	"github.com/zyy125/im-system/internal/ws"
	"github.com/zyy125/im-system/internal/mq"
)

type App struct {
	Router *gin.Engine
}

func InitApp(cfg *config.Config, ctx context.Context) (*App, error) {
	db, err := repository.NewMysql(cfg.Mysql.DSN)
	if err != nil {
		return nil, err
	}
	rdb, err := repository.NewRedisClient(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return nil, err
	}

	userRepo := repository.NewUserRepo(db)
	blacklistRepo := repository.NewTokenBlacklistRepo(rdb)
	presenceRepo := repository.NewPresenceRepo(rdb)
	msgRepo := repository.NewMessageRepo(db)

	hub := ws.NewHub(presenceRepo)
	msgSvc := service.NewMessageService(msgRepo)
	rabbitMQ, err := mq.NewRabbitMQ(cfg.RabbitMQ.URL, msgSvc)
	if err != nil {
		return nil, err
	}
	wsHandler := handler.NewWsHandler(hub, rabbitMQ)

	// start rabbitmq consumer
	if err := rabbitMQ.ConsumeChatMsg(ctx); err != nil {
		return nil, err
	}

	go hub.Run(ctx)

	authSvc := service.NewAuthService(userRepo, &cfg.JWT, blacklistRepo)
	authHandler := handler.NewAuthHandler(authSvc)

	userSvc := service.NewUserService(userRepo, presenceRepo)
	userHandler := handler.NewUserHandler(userSvc)

	initRouterParams := &router.InitRouterParams{
		AuthHandler: authHandler,
		WsHandler: wsHandler,
		UserHandler: userHandler,

		BlacklistRepo: blacklistRepo,
		JwtCfg: &cfg.JWT,
	}
	router := router.InitRouter(initRouterParams)

	return &App{Router: router}, nil
}