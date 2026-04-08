package ws

import (
	"context"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zyy125/im-system/internal/apperr"
)

var (
	maxMessageSize    int64 = 2048
	pongWait                = 60 * time.Second
	writeWait               = 10 * time.Second
	heartbeatInterval       = 30 * time.Second
)

type Client struct {
	UserID      uint64          `json:"user_id"`
	Conn        *websocket.Conn `json:"conn"`
	Send        chan []byte     `json:"send"`
	Hub         *Hub            `json:"hub"`
	ChatHandler ChatSendHandler `json:"-"`
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

		req, err := DecodeClientChatSend(message)
		if err != nil {
			log.Printf("Client %d decode client message error: %v", c.UserID, err)
			c.writeError(err)
			continue
		}
		if c.ChatHandler == nil {
			err := apperr.Internal("chat handler unavailable", nil)
			log.Printf("Client %d chat handler unavailable", c.UserID)
			c.writeError(err)
			continue
		}
		forwardMsg, err := c.ChatHandler.HandleChatSend(ctx, c.UserID, req)
		if err != nil {
			log.Printf("Client %d handle chat send error: %v", c.UserID, err)
			c.writeError(err)
			continue
		}

		select {
		case c.Hub.Forward <- forwardMsg:
		default:
			log.Printf("Client %d forward message dropped", c.UserID)
		}
	}
}

func (c *Client) writeError(err error) {
	appErr := apperr.From(err)
	payload, marshalErr := MarshalEnvelope(EventTypeError, ErrorData{
		Code:    string(appErr.Code),
		Message: appErr.Message,
	})
	if marshalErr != nil {
		log.Printf("Client %d marshal error payload failed: %v", c.UserID, marshalErr)
		return
	}
	select {
	case c.Send <- payload:
	default:
		log.Printf("Client %d error payload dropped: send queue full", c.UserID)
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
				_ = c.writeMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.writeMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Client %d write message error: %v", c.UserID, err)
				return
			}

		case <-ticker.C:
			if err := c.writeMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Printf("Client %d write ping message error: %v", c.UserID, err)
				return
			}
		}
	}
}

func (c *Client) writeMessage(messageType int, payload []byte) error {
	if c.Conn == nil {
		return nil
	}
	if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return c.Conn.WriteMessage(messageType, payload)
}
