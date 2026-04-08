package ws

import (
	"encoding/json"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
)

const ProtocolVersion = 1

type Envelope struct {
	Type    string          `json:"type"`
	Version int             `json:"version"`
	Data    json.RawMessage `json:"data"`
}

type outboundEnvelope struct {
	Type    string `json:"type"`
	Version int    `json:"version"`
	Data    any    `json:"data"`
}

type ClientChatSend struct {
	MsgID    string `json:"msg_id"`
	To       uint64 `json:"to"`
	Content  string `json:"content"`
	SendTime int64  `json:"send_time,omitempty"`
}

type ServerChatMessage struct {
	ID             uint64 `json:"id"`
	MsgID          string `json:"msg_id"`
	ConversationID string `json:"conversation_id"`
	From           uint64 `json:"from"`
	To             uint64 `json:"to"`
	SendTime       int64  `json:"send_time"`
	Content        string `json:"content"`
}

type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PresenceChangedData struct {
	UserID uint64 `json:"user_id"`
	Online bool   `json:"online"`
}

func MarshalEnvelope(eventType string, data any) ([]byte, error) {
	return json.Marshal(outboundEnvelope{
		Type:    eventType,
		Version: ProtocolVersion,
		Data:    data,
	})
}

func DecodeClientChatSend(payload []byte) (ClientChatSend, error) {
	var env Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return ClientChatSend{}, apperr.MessageInvalidPayload()
	}
	if env.Type != EventTypeChatSend || len(env.Data) == 0 {
		return ClientChatSend{}, apperr.MessageInvalidPayload()
	}

	var req ClientChatSend
	if err := json.Unmarshal(env.Data, &req); err != nil {
		return ClientChatSend{}, apperr.MessageInvalidPayload()
	}
	return req, nil
}

func NewServerChatMessage(msg model.ChatMessage) ServerChatMessage {
	return ServerChatMessage{
		ID:             msg.ID,
		MsgID:          msg.MsgID,
		ConversationID: msg.ConversationID,
		From:           msg.From,
		To:             msg.To,
		SendTime:       msg.SendTime,
		Content:        msg.Content,
	}
}
