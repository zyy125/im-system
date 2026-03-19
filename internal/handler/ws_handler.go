package handler

import(
	"net/http"
	"log"


	"github.com/zyy125/im-system/pkg/response"
	"github.com/zyy125/im-system/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zyy125/im-system/internal/mq"
)

type WsHandler struct {
	hub *ws.Hub
	mq  *mq.RabbitMQ
}

func NewWsHandler(hub *ws.Hub, mq *mq.RabbitMQ) *WsHandler {
	return &WsHandler{hub: hub, mq: mq}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *WsHandler) HandleWs(c *gin.Context) {
	userID := c.GetUint64("userID")
	log.Printf("Client %d connect", userID)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ctx := c.Request.Context()

	client := &ws.Client{
		UserID: userID,
		Conn: conn,
		Ctx: ctx,
		Send: make(chan []byte, 256),
		Hub: h.hub,
		Mq: h.mq,
	}

	h.hub.Register <- client

	go client.WritePump(ctx)
	go client.ReadPump(ctx)
}