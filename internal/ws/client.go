package ws

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/mq"
)

var (
	maxMessageSize    int64 = 2048
	pongWait                = 60 * time.Second
	writeWait               = 10 * time.Second
	heartbeatInterval       = 30 * time.Second
)

type Client struct {
	UserID uint64          `json:"user_id"`
	Conn   *websocket.Conn `json:"conn"`
	Send   chan []byte     `json:"send"`
	Hub    *Hub            `json:"hub"`
	Ctx    context.Context `json:"ctx"`
	Mq     *mq.RabbitMQ    `json:"mq"`
}

func (c *Client) ReadPump(ctx context.Context) {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Client %d read message error: %v", c.UserID, err)
			break
		}
		//异步存入RabbitMQ
		go func() {
			if err = c.Mq.PublishChatMsg(ctx, message); err != nil {
				log.Printf("Client %d publish message error: %v", c.UserID, err)
			}
		}()

		var chatMsg model.ChatMsg
		if err = json.Unmarshal(message, &chatMsg); err != nil {
			log.Printf("Client %d unmarshal message error: %v", c.UserID, err)
			continue
		}
		select {
		case c.Hub.Forward <- &ForwardMsg{
			To:      chatMsg.To,
			From:    c.UserID,
			Content: []byte(chatMsg.Content),
		}:
		default:
			log.Printf("Client %d forward message error: %v", c.UserID, err)
		}
	}
}

func (c *Client) WritePump(ctx context.Context) {
	defer func() {
		c.Conn.Close()
	}()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Client %d write message error: %v", c.UserID, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Printf("Client %d write ping message error: %v", c.UserID, err)
				return
			}
		}
	}
}
