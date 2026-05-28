package app

import (
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

// repositories 聚合所有数据访问层依赖，便于在装配函数间传递。
type repositories struct {
	userRepo           repository.UserRepo
	blacklistRepo      repository.TokenBlacklistRepo
	refreshSessionRepo repository.RefreshSessionRepo
	presenceRepo       repository.PresenceRepo
	msgRepo            repository.MessageRepo
	friendRepo         repository.FriendRepo
	friendRequestRepo  repository.FriendRequestRepo
	conversationRepo   repository.ConversationRepo
	messageTxManager   repository.MessageTxManager
	messageStateRepo   repository.MessageStateRepo
	redisClient        *redis.Client
}

// services 聚合所有业务逻辑层依赖。
type services struct {
	authSvc          service.AuthService
	userSvc          service.UserService
	friendSvc        service.FriendService
	friendRequestSvc service.FriendRequestService
	messageSvc       service.MessageService
	messageSendSvc   service.MessageSendService
	conversationSvc  service.ConversationService
}

// realtimeComponents 聚合实时通信相关组件。
type realtimeComponents struct {
	hub *ws.Hub
}

// handlers 聚合所有 HTTP/WebSocket 处理器。
type handlers struct {
	authHandler          *handler.AuthHandler
	avatarHandler        *handler.AvatarHandler
	debugHandler         *handler.DebugHandler
	wsHandler            *handler.WSHandler
	userHandler          *handler.UserHandler
	friendHandler        *handler.FriendHandler
	friendRequestHandler *handler.FriendRequestHandler
	messageHandler       *handler.MessageHandler
	conversationHandler  *handler.ConversationHandler
}

func initRepositories(cfg *config.Config, db *gorm.DB, rdb *redis.Client) *repositories {
	return &repositories{
		userRepo:           repository.NewUserRepo(db),
		blacklistRepo:      repository.NewTokenBlacklistRepo(rdb),
		refreshSessionRepo: repository.NewRefreshSessionRepo(rdb),
		presenceRepo:       repository.NewPresenceRepo(rdb, cfg.Presence.TTL),
		msgRepo:            repository.NewMessageRepo(db),
		friendRepo:         repository.NewFriendRepo(db),
		friendRequestRepo:  repository.NewFriendRequestRepo(db),
		conversationRepo:   repository.NewConversationRepo(db),
		messageTxManager:   repository.NewMessageTxManager(db),
		messageStateRepo:   repository.NewMessageStateRepo(rdb),
		redisClient:        rdb,
	}
}

func initServices(cfg *config.Config, repos *repositories) *services {
	seqAllocator := service.NewSeqAllocator(repos.msgRepo, repos.messageStateRepo)
	messageSvc := service.NewMessageService(repos.msgRepo, repos.conversationRepo)
	messageSendSvc := service.NewMessageSendService(repos.messageTxManager, seqAllocator)
	conversationSvc := service.NewConversationServiceWithRuntime(
		repos.conversationRepo,
		repos.msgRepo,
		repos.userRepo,
		repos.presenceRepo,
		repos.messageTxManager,
		seqAllocator,
	)
	friendSvc := service.NewFriendService(
		repos.friendRepo,
		repos.userRepo,
		repos.presenceRepo,
		repos.conversationRepo,
	)

	return &services{
		authSvc:          service.NewAuthService(repos.userRepo, &cfg.JWT, repos.blacklistRepo, repos.refreshSessionRepo),
		userSvc:          service.NewUserService(repos.userRepo, repos.presenceRepo),
		friendSvc:        friendSvc,
		friendRequestSvc: service.NewFriendRequestService(repos.friendRequestRepo, friendSvc, repos.userRepo, repos.presenceRepo),
		messageSvc:       messageSvc,
		messageSendSvc:   messageSendSvc,
		conversationSvc:  conversationSvc,
	}
}

func initRealtime(cfg *config.Config, repos *repositories, svcs *services) (*realtimeComponents, error) {
	hub := ws.NewHub(repos.presenceRepo, svcs.conversationSvc, repos.friendRepo, svcs.userSvc)
	return &realtimeComponents{
		hub: hub,
	}, nil
}

func initHandlers(cfg *config.Config, repos *repositories, rt *realtimeComponents, svcs *services) *handlers {
	return &handlers{
		authHandler:          handler.NewAuthHandler(svcs.authSvc),
		avatarHandler:        handler.NewAvatarHandler(svcs.userSvc),
		debugHandler:         handler.NewDebugHandler(rt.hub),
		wsHandler:            handler.NewWSHandler(rt.hub, svcs.messageSendSvc, svcs.messageSvc, svcs.conversationSvc, svcs.userSvc, cfg.JWT.Secret, repos.blacklistRepo, cfg.WS, cfg.App.Env),
		userHandler:          handler.NewUserHandler(svcs.userSvc),
		friendHandler:        handler.NewFriendHandler(svcs.friendSvc, svcs.userSvc),
		friendRequestHandler: handler.NewFriendRequestHandler(svcs.friendRequestSvc, svcs.userSvc),
		messageHandler:       handler.NewMessageHandler(svcs.messageSvc, svcs.conversationSvc, svcs.userSvc),
		conversationHandler:  handler.NewConversationHandler(svcs.conversationSvc, svcs.userSvc),
	}
}

func buildRouter(hs *handlers, repos *repositories, cfg *config.Config) *gin.Engine {
	return router.InitRouter(&router.InitRouterParams{
		AuthHandler:          hs.authHandler,
		AvatarHandler:        hs.avatarHandler,
		DebugHandler:         hs.debugHandler,
		WSHandler:            hs.wsHandler,
		UserHandler:          hs.userHandler,
		FriendHandler:        hs.friendHandler,
		FriendRequestHandler: hs.friendRequestHandler,
		MessageHandler:       hs.messageHandler,
		ConversationHandler:  hs.conversationHandler,
		BlacklistRepo:        repos.blacklistRepo,
		AppCfg:               &cfg.App,
		HTTPCfg:              &cfg.HTTP,
		JwtCfg:               &cfg.JWT,
		StorageCfg:           &cfg.Storage,
	})
}
