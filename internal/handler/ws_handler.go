package handler

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/logging"
	"github.com/zyy125/im-system/internal/middleware"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/internal/service"
	"github.com/zyy125/im-system/internal/ws"
	"github.com/zyy125/im-system/pkg/utils"
)

type WSHandler struct {
	hub                     *ws.Hub
	messageSendService      service.MessageCommandService
	messageReceiptService   service.MessageReceiptService
	conversationSyncService service.ConversationSyncService
	jwtSecret               string
	tokenBlacklistRepo      repository.TokenBlacklistRepo
	allowedOrigins          []string
	allowAnyOrigin          bool
}

func NewWSHandler(
	hub *ws.Hub,
	messageSendService service.MessageCommandService,
	messageReceiptService service.MessageReceiptService,
	conversationSyncService service.ConversationSyncService,
	jwtSecret string,
	tokenBlacklistRepo repository.TokenBlacklistRepo,
	wsCfg config.WS,
	appEnv string,
) *WSHandler {
	return &WSHandler{
		hub:                     hub,
		messageSendService:      messageSendService,
		messageReceiptService:   messageReceiptService,
		conversationSyncService: conversationSyncService,
		jwtSecret:               jwtSecret,
		tokenBlacklistRepo:      tokenBlacklistRepo,
		allowedOrigins:          normalizeOrigins(wsCfg.AllowedOrigins),
		allowAnyOrigin:          strings.TrimSpace(appEnv) != "production" && len(wsCfg.AllowedOrigins) == 0,
	}
}

// HandleWS 建立 WebSocket 连接
// @Summary 建立 WebSocket 连接
// @Description 建立当前用户的 WebSocket 长连接，用于实时消息与在线状态推送。优先使用 `Authorization: Bearer <token>`；浏览器场景可使用 `?token=<jwt>`。生产环境会按 `ws.allowed_origins` 校验 Origin。
// @Tags WebSocket
// @Produce plain
// @Security BearerAuth
// @Param Authorization header string false "Bearer JWT，优先使用"
// @Param token query string false "仅限 WebSocket 握手使用的 JWT query token"
// @Success 101 {string} string "Switching Protocols"
// @Failure 401 {object} response.Response "未认证"
// @Failure 403 {object} response.Response "Origin 不在允许列表中"
// @Failure 500 {object} response.Response "升级连接失败"
// @Router /api/v1/ws/ [get]
func (h *WSHandler) HandleWS(c *gin.Context) {
	authResult, err := middleware.AuthenticateWSRequest(c.Request, h.jwtSecret, h.tokenBlacklistRepo)
	if err != nil {
		respondError(c, err)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: h.checkOrigin,
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		respondError(c, apperr.Internal("upgrade websocket connection failed", err))
		return
	}

	connectionID := utils.GenerateUUID()
	logger := logging.With(
		"user_id", authResult.UserID,
		"connection_id", connectionID,
		"event_type", "ws_connect",
	)
	logger.Info("websocket client connected")

	ctx := logging.ContextWithLogger(context.Background(), logger)

	client := &ws.Client{
		ConnectionID: connectionID,
		UserID:       authResult.UserID,
		Conn:         conn,
		Send:         make(chan []byte, 256),
		Hub:          h.hub,
		ChatHandler:  ws.NewChatSendHandler(h.messageSendService),
		AckHandler:   ws.NewMessageAckHandler(h.messageReceiptService, h.conversationSyncService),
	}

	go client.WritePump(ctx)
	h.hub.Register <- client

	go client.ReadPump(ctx)
}

func (h *WSHandler) checkOrigin(r *http.Request) bool {
	if h.allowAnyOrigin {
		return true
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return false
	}
	return slices.Contains(h.allowedOrigins, origin)
}

func normalizeOrigins(origins []string) []string {
	items := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		items = append(items, origin)
	}
	return items
}
