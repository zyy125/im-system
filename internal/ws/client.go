package ws

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/logging"
)

var (
	maxMessageSize    int64 = 2048
	pongWait                = 60 * time.Second
	writeWait               = 10 * time.Second
	heartbeatInterval       = 30 * time.Second
)

type Client struct {
	ConnectionID string            `json:"connection_id"`
	UserID       uint64            `json:"user_id"`
	Conn         *websocket.Conn   `json:"conn"`
	Send         chan []byte       `json:"send"`
	Hub          *Hub              `json:"hub"`
	ChatHandler  ChatSendHandler   `json:"-"`
	AckHandler   MessageAckHandler `json:"-"`
	Lifecycle    ClientLifecycle   `json:"-"`
	closed       atomic.Bool
}

func (c *Client) ReadPump(ctx context.Context) {
	defer func() {
		if c.Hub != nil {
			c.Hub.EnqueueUnregisterWithReason(c, CloseReasonClientReadStopped)
		}
		closeClientConn(c)
	}()

	logger := logging.FromContext(ctx)
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		if c.Lifecycle != nil {
			c.Lifecycle.Refresh(ctx, c.UserID)
		}
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			logger.Info("read pump stopped", "error", err)
			break
		}

		env, err := DecodeEnvelope(message)
		if err != nil {
			logger.Warn("decode client envelope failed", "error", err)
			c.writeError(err)
			continue
		}

		switch env.Type {
		case EventTypeMessageSend:
			req, err := DecodeClientMessageSendData(env.Data)
			if err != nil {
				logger.Warn("decode message send payload failed", "error", err)
				c.writeError(err)
				continue
			}
			if c.ChatHandler == nil {
				err := apperr.Internal("chat handler unavailable", nil)
				logger.Error("chat handler unavailable")
				c.writeError(err)
				continue
			}
			forwardMsgs, err := c.ChatHandler.HandleMessageSend(ctx, c.UserID, req)
			if err != nil {
				logger.Warn("handle message send failed", "conversation_id", req.ConversationID, "msg_id", req.MsgID, "error", err)
				c.writeError(err)
				continue
			}
			c.Hub.EnqueueForwards(ctx, forwardMsgs)

		case EventTypeMessageDelivered:
			req, err := DecodeClientMessageDeliveredData(env.Data)
			if err != nil {
				logger.Warn("decode message delivered payload failed", "error", err)
				c.writeError(err)
				continue
			}
			if c.AckHandler == nil {
				err := apperr.Internal("receipt handler unavailable", nil)
				logger.Error("receipt handler unavailable")
				c.writeError(err)
				continue
			}
			forwardMsgs, err := c.AckHandler.HandleMessageDelivered(ctx, c.UserID, req)
			if err != nil {
				logger.Warn("handle message delivered failed", "conversation_id", req.ConversationID, "error", err)
				c.writeError(err)
				continue
			}
			c.Hub.EnqueueForwards(ctx, forwardMsgs)

		case EventTypeMessageRead:
			req, err := DecodeClientMessageReadData(env.Data)
			if err != nil {
				logger.Warn("decode message read payload failed", "error", err)
				c.writeError(err)
				continue
			}
			if c.AckHandler == nil {
				err := apperr.Internal("receipt handler unavailable", nil)
				logger.Error("receipt handler unavailable")
				c.writeError(err)
				continue
			}
			forwardMsgs, err := c.AckHandler.HandleMessageRead(ctx, c.UserID, req)
			if err != nil {
				logger.Warn("handle message read failed", "conversation_id", req.ConversationID, "error", err)
				c.writeError(err)
				continue
			}
			c.Hub.EnqueueForwards(ctx, forwardMsgs)

		default:
			err := apperr.MessageInvalidPayload()
			logger.Warn("received unsupported ws event", "event_type", env.Type)
			c.writeError(err)
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
		logging.With("user_id", c.UserID, "connection_id", c.ConnectionID).Error("marshal error payload failed", "error", marshalErr)
		return
	}
	if !c.enqueuePayload(payload) {
		logging.With("user_id", c.UserID, "connection_id", c.ConnectionID).Warn("error payload dropped: send queue full")
	}
}

func (c *Client) WritePump(ctx context.Context) {
	defer func() {
		c.Conn.Close()
	}()

	logger := logging.FromContext(ctx)
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-c.Send:
			if !ok {
				_ = c.writeMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.writeMessage(websocket.TextMessage, message); err != nil {
				logger.Info("write message failed", "error", err)
				return
			}

		case <-ticker.C:
			if err := c.writeMessage(websocket.PingMessage, []byte{}); err != nil {
				logger.Info("write ping failed", "error", err)
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

func (c *Client) enqueuePayload(payload []byte) (ok bool) {
	if c == nil || c.Send == nil || c.closed.Load() {
		return false
	}

	defer func() {
		if recover() != nil {
			ok = false
		}
	}()

	select {
	case c.Send <- payload:
		return true
	default:
		return false
	}
}
