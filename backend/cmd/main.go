// @title IM System API
// @version 1.0
// @description 单体 IM 后端 API。HTTP 受保护接口只接受 `Authorization: Bearer <token>`；WebSocket 握手接受 `Authorization: Bearer <token>`，并仅为浏览器兼容保留 `?token=<jwt>`。生产环境要求显式配置 `ws.allowed_origins`，运行配置新增 `app.http_addr`、`ws.allowed_origins`、`presence.ttl`、`presence.heartbeat_interval`。
// @host localhost:8080
// @BasePath /

// JWT认证
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description HTTP 受保护接口必须使用 Bearer Token；WebSocket 握手也支持该 Header。

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/app"
	"github.com/zyy125/im-system/internal/logging"
)

func main() {
	logging.Setup()
	if err := run(); err != nil {
		slog.Error("application exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return app.Run(ctx, cfg)
}
