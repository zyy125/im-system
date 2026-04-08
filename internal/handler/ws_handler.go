package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zyy125/im-system/internal/service"
	"github.com/zyy125/im-system/internal/ws"
)

type WSHandler struct {
	hub                 *ws.Hub
	messageService      service.MessageService
	friendService       service.FriendService
	conversationService service.ConversationService
}

func NewWSHandler(
	hub *ws.Hub,
	messageService service.MessageService,
	friendService service.FriendService,
	conversationService service.ConversationService,
) *WSHandler {
	return &WSHandler{
		hub:                 hub,
		messageService:      messageService,
		friendService:       friendService,
		conversationService: conversationService,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// HandleWS 建立 WebSocket 连接
// @Summary 建立 WebSocket 连接
// @Description 建立当前用户的 WebSocket 长连接，用于实时消息与在线状态推送
// @Tags WebSocket
// @Produce plain
// @Security BearerAuth
// @Success 101 {string} string "Switching Protocols"
// @Failure 401 {object} response.Response "未认证"
// @Failure 500 {object} response.Response "升级连接失败"
// @Router /api/v1/ws/ [get]
func (h *WSHandler) HandleWS(c *gin.Context) {
	userID := currentUserID(c)
	log.Printf("Client %d connect", userID)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		respondError(c, err)
		return
	}
	// WebSocket 连接是长连接，c.Request.Context() 在 Upgrade 完成后就会被取消
	// 这里使用 context.Background()，并在 client 结构中统一管理
	ctx := context.Background()

	client := &ws.Client{
		UserID:      userID,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		Hub:         h.hub,
		ChatHandler: ws.NewChatSendHandler(h.messageService, h.friendService, h.conversationService),
	}

	go client.WritePump(ctx)
	h.hub.Register <- client

	go client.ReadPump(ctx)
}
