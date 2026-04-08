package router

import (
	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/handler"
	"github.com/zyy125/im-system/internal/middleware"
	"github.com/zyy125/im-system/internal/repository"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/zyy125/im-system/docs" // 生成的swagger文档
)

type InitRouterParams struct {
	AuthHandler          *handler.AuthHandler
	WSHandler            *handler.WSHandler
	UserHandler          *handler.UserHandler
	FriendHandler        *handler.FriendHandler
	FriendRequestHandler *handler.FriendRequestHandler
	MessageHandler       *handler.MessageHandler
	ConversationHandler  *handler.ConversationHandler

	BlacklistRepo repository.TokenBlacklistRepo
	JwtCfg        *config.JWT
}

func InitRouter(params *InitRouterParams) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.ErrorSourceLogger())

	// Swagger 文档
	if gin.Mode() != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	authHandler := params.AuthHandler
	blacklistRepo := params.BlacklistRepo
	jwtCfg := params.JwtCfg
	wsHandler := params.WSHandler
	userHandler := params.UserHandler
	friendHandler := params.FriendHandler
	friendRequestHandler := params.FriendRequestHandler
	messageHandler := params.MessageHandler
	conversationHandler := params.ConversationHandler

	r.StaticFile("/", "./web/index.html")
	r.Static("/web", "./web")

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", middleware.AuthMiddleware(jwtCfg.Secret, blacklistRepo), authHandler.Logout)
		}

		user := api.Group("/user", middleware.AuthMiddleware(jwtCfg.Secret, blacklistRepo))
		{
			user.GET("/online", userHandler.CheckUserOnline)
		}

		users := api.Group("/users", middleware.AuthMiddleware(jwtCfg.Secret, blacklistRepo))
		{
			users.GET("/me", userHandler.GetMe)
			users.GET("/:id", userHandler.GetUserInfo)
		}

		friends := api.Group("/friends", middleware.AuthMiddleware(jwtCfg.Secret, blacklistRepo))
		{
			friends.DELETE("/:id", friendHandler.RemoveFriend)
			friends.GET("", friendHandler.ListFriends)
		}

		friendRequests := api.Group("/friend-requests", middleware.AuthMiddleware(jwtCfg.Secret, blacklistRepo))
		{
			friendRequests.POST("/:id", friendRequestHandler.Send)
			friendRequests.GET("/incoming", friendRequestHandler.Incoming)
			friendRequests.GET("/outgoing", friendRequestHandler.Outgoing)
			friendRequests.POST("/:id/accept", friendRequestHandler.Accept)
			friendRequests.POST("/:id/reject", friendRequestHandler.Reject)
		}

		messages := api.Group("/messages", middleware.AuthMiddleware(jwtCfg.Secret, blacklistRepo))
		{
			messages.GET("/history", messageHandler.History)
			messages.POST("/read", messageHandler.MarkRead)
		}

		conversations := api.Group("/conversations", middleware.AuthMiddleware(jwtCfg.Secret, blacklistRepo))
		{
			conversations.GET("", conversationHandler.List)
			conversations.POST("/direct/:id/open", conversationHandler.OpenDirect)
			conversations.POST("/:id/hide", conversationHandler.Hide)
		}

		ws := api.Group("/ws", middleware.AuthMiddleware(jwtCfg.Secret, blacklistRepo))
		{
			ws.GET("/", wsHandler.HandleWS)
		}
	}

	return r
}
