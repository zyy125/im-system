package app

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/infra"
)

type App struct {
	Router *gin.Engine
}

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

	rt, err := initRealtime(repos, svcs)
	if err != nil {
		return nil, err
	}
	if err := startRealtime(ctx, rt); err != nil {
		return nil, err
	}

	hs := initHandlers(rt, svcs)
	return &App{Router: buildRouter(hs, repos, cfg)}, nil
}
