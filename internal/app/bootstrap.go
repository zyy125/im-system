package app

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/handler"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/internal/router"
	"github.com/zyy125/im-system/internal/service"
	"github.com/zyy125/im-system/internal/ws"
	"gorm.io/gorm"
)

type repositories struct {
	userRepo          repository.UserRepo
	blacklistRepo     repository.TokenBlacklistRepo
	presenceRepo      repository.PresenceRepo
	msgRepo           repository.MessageRepo
	friendRepo        repository.FriendRepo
	friendRequestRepo repository.FriendRequestRepo
	conversationRepo  repository.ConversationRepo
	messageTxManager  repository.MessageTxManager
}

type services struct {
	authSvc          service.AuthService
	userSvc          service.UserService
	friendSvc        service.FriendService
	friendRequestSvc service.FriendRequestService
	messageSvc       service.MessageService
	conversationSvc  service.ConversationService
}

type realtimeComponents struct {
	hub *ws.Hub
}

type handlers struct {
	authHandler          *handler.AuthHandler
	wsHandler            *handler.WSHandler
	userHandler          *handler.UserHandler
	friendHandler        *handler.FriendHandler
	friendRequestHandler *handler.FriendRequestHandler
	messageHandler       *handler.MessageHandler
	conversationHandler  *handler.ConversationHandler
}

func initRepositories(db *gorm.DB, rdb *redis.Client) *repositories {
	return &repositories{
		userRepo:          repository.NewUserRepo(db),
		blacklistRepo:     repository.NewTokenBlacklistRepo(rdb),
		presenceRepo:      repository.NewPresenceRepo(rdb),
		msgRepo:           repository.NewMessageRepo(db),
		friendRepo:        repository.NewFriendRepo(db),
		friendRequestRepo: repository.NewFriendRequestRepo(db),
		conversationRepo:  repository.NewConversationRepo(db),
		messageTxManager:  repository.NewMessageTxManager(db),
	}
}

func initServices(cfg *config.Config, repos *repositories) *services {
	conversationSvc := service.NewConversationService(
		repos.conversationRepo,
		repos.msgRepo,
		repos.userRepo,
		repos.presenceRepo,
		repos.friendRepo,
	)
	friendSvc := service.NewFriendService(
		repos.friendRepo,
		repos.userRepo,
		repos.presenceRepo,
		repos.conversationRepo,
	)

	return &services{
		authSvc:          service.NewAuthService(repos.userRepo, &cfg.JWT, repos.blacklistRepo),
		userSvc:          service.NewUserService(repos.userRepo, repos.presenceRepo),
		friendSvc:        friendSvc,
		friendRequestSvc: service.NewFriendRequestService(repos.friendRequestRepo, friendSvc, repos.userRepo, repos.presenceRepo),
		messageSvc:       service.NewMessageService(repos.msgRepo, repos.conversationRepo, repos.messageTxManager),
		conversationSvc:  conversationSvc,
	}
}

func initRealtime(repos *repositories, svcs *services) (*realtimeComponents, error) {
	hub := ws.NewHub(repos.presenceRepo, svcs.conversationSvc, repos.friendRepo)
	return &realtimeComponents{
		hub: hub,
	}, nil
}

func initHandlers(rt *realtimeComponents, svcs *services) *handlers {
	return &handlers{
		authHandler:          handler.NewAuthHandler(svcs.authSvc),
		wsHandler:            handler.NewWSHandler(rt.hub, svcs.messageSvc, svcs.friendSvc, svcs.conversationSvc),
		userHandler:          handler.NewUserHandler(svcs.userSvc),
		friendHandler:        handler.NewFriendHandler(svcs.friendSvc),
		friendRequestHandler: handler.NewFriendRequestHandler(svcs.friendRequestSvc),
		messageHandler:       handler.NewMessageHandler(svcs.messageSvc, svcs.friendSvc, svcs.conversationSvc),
		conversationHandler:  handler.NewConversationHandler(svcs.conversationSvc),
	}
}

func buildRouter(hs *handlers, repos *repositories, cfg *config.Config) *gin.Engine {
	return router.InitRouter(&router.InitRouterParams{
		AuthHandler:          hs.authHandler,
		WSHandler:            hs.wsHandler,
		UserHandler:          hs.userHandler,
		FriendHandler:        hs.friendHandler,
		FriendRequestHandler: hs.friendRequestHandler,
		MessageHandler:       hs.messageHandler,
		ConversationHandler:  hs.conversationHandler,
		BlacklistRepo:        repos.blacklistRepo,
		JwtCfg:               &cfg.JWT,
	})
}

func startRealtime(ctx context.Context, rt *realtimeComponents) error {
	go rt.hub.Run(ctx)
	return nil
}
