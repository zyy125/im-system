package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/infra"
	"github.com/zyy125/im-system/internal/logging"
	"github.com/zyy125/im-system/internal/ws"
)

type App struct {
	Router      *gin.Engine
	Server      *http.Server
	Hub         *ws.Hub
	RedisClient *redis.Client
	SQLDB       *sql.DB

	hubCancel    context.CancelFunc
	shutdownOnce sync.Once
}

func Run(ctx context.Context, cfg *config.Config) error {
	app, err := InitApp(cfg, ctx)
	if err != nil {
		return err
	}
	return app.Start(ctx)
}

func (a *App) Start(ctx context.Context) error {
	if a == nil || a.Server == nil {
		return errors.New("app server is not initialized")
	}

	hubCtx, hubCancel := context.WithCancel(context.Background())
	a.hubCancel = hubCancel
	if a.Hub != nil {
		go a.Hub.Run(hubCtx)
	}

	errCh := make(chan error, 1)
	go func() {
		err := a.Server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		if err != nil {
			_ = a.Shutdown(context.Background())
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return a.Shutdown(shutdownCtx)
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	if a == nil {
		return nil
	}

	var shutdownErr error
	a.shutdownOnce.Do(func() {
		if a.hubCancel != nil {
			a.hubCancel()
		}
		if a.Hub != nil {
			select {
			case <-a.Hub.Done():
			case <-ctx.Done():
				if shutdownErr == nil {
					shutdownErr = ctx.Err()
				}
			}
		}
		if a.Server != nil {
			if err := a.Server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				shutdownErr = err
			}
		}
		if a.RedisClient != nil {
			if err := a.RedisClient.Close(); err != nil && shutdownErr == nil {
				shutdownErr = err
			}
		}
		if a.SQLDB != nil {
			if err := a.SQLDB.Close(); err != nil && shutdownErr == nil {
				shutdownErr = err
			}
		}
	})
	return shutdownErr
}

func InitApp(cfg *config.Config, ctx context.Context) (*App, error) {
	db, err := infra.NewMySQL(cfg.Mysql.DSN)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	rdb, err := infra.NewRedisClient(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return nil, err
	}

	repos := initRepositories(cfg, db, rdb)
	svcs := initServices(cfg, repos)

	rt, err := initRealtime(cfg, repos, svcs)
	if err != nil {
		return nil, err
	}
	hs := initHandlers(cfg, repos, rt, svcs)
	engine := buildRouter(hs, repos, cfg)

	server := &http.Server{
		Addr:    cfg.App.HTTPAddr,
		Handler: engine,
	}

	logging.With("http_addr", cfg.App.HTTPAddr, "env", cfg.App.Env).Info("application initialized")

	return &App{
		Router:      engine,
		Server:      server,
		Hub:         rt.hub,
		RedisClient: rdb,
		SQLDB:       sqlDB,
	}, nil
}
