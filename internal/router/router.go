package router

import (
	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/handler"
	"github.com/zyy125/im-system/internal/middleware"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/config"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/zyy125/im-system/docs" // 生成的swagger文档
)

type InitRouterParams struct {
	AuthHandler *handler.AuthHandler
	WsHandler *handler.WsHandler
	UserHandler *handler.UserHandler
		
	BlacklistRepo repository.TokenBlacklistRepo
	JwtCfg *config.JWT
}

func InitRouter(params *InitRouterParams) *gin.Engine {
	r := gin.Default()

	// Swagger 文档
	if gin.Mode() != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	authHandler := params.AuthHandler
	blacklistRepo := params.BlacklistRepo
	jwtCfg := params.JwtCfg
	wsHandler := params.WsHandler
	userHandler := params.UserHandler
		
	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout, middleware.AuthMiddleware(jwtCfg.Secret, blacklistRepo))
		}

		user := api.Group("/user", middleware.AuthMiddleware(jwtCfg.Secret, blacklistRepo))
		{
			user.GET("/online", userHandler.CheckUserOnline)
		}

		ws := api.Group("/ws", middleware.AuthMiddleware(jwtCfg.Secret, blacklistRepo))
		{
			ws.GET("/", wsHandler.HandleWs)
		}
	}
	
	return r
}