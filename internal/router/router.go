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

	userHandler := params.UserHandler
	blacklistRepo := params.BlacklistRepo
	jwtCfg := params.JwtCfg

	api := r.Group("/api")
	{
		users := api.Group("/users")
		{
			users.POST("/register", userHandler.Register)
			users.POST("/login", userHandler.Login)

			auth := users.Group("/auth", middleware.AuthMiddleware(jwtCfg.Secret, blacklistRepo))
			{
				auth.POST("/logout", userHandler.Logout)
			}
		}
	}
	
	return r
}