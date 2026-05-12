package app

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/infra"
)

// App 是应用入口，目前只暴露 Gin 路由引擎供外部调用 Run。
type App struct {
	Router *gin.Engine
}

const defaultHTTPAddr = ":8080"

// Run 初始化 API 进程并启动 HTTP 服务。
func Run(ctx context.Context, cfg *config.Config) error {
	app, err := InitApp(cfg, ctx)
	if err != nil {
		return err
	}
	return app.Run(defaultHTTPAddr)
}

// Run 启动当前应用实例的 HTTP 服务。
func (a *App) Run(addr string) error {
	return a.Router.Run(addr)
}

// InitApp 按顺序完成以下装配步骤：
//  1. 连接 MySQL 和 Redis
//  2. 初始化所有 repository
//  3. 初始化所有 service
//  4. 初始化实时组件（Hub）并启动后台 goroutine
//  5. 初始化 handler 并构建 Gin 路由
func InitApp(cfg *config.Config, ctx context.Context) (*App, error) {
	db, err := infra.NewMySQL(cfg.Mysql.DSN)
	if err != nil {
		return nil, err
	}
	rdb, err := infra.NewRedisClient(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return nil, err
	}

	repos := initRepositories(db, rdb)
	svcs := initServices(cfg, repos)

	rt, err := initRealtime(cfg, repos, svcs)
	if err != nil {
		return nil, err
	}
	if err := startRealtime(ctx, rt); err != nil {
		return nil, err
	}

	hs := initHandlers(rt, svcs)
	return &App{Router: buildRouter(hs, repos, cfg)}, nil
}
