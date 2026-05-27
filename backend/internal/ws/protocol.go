package ws

import (
	"encoding/json"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
)

type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type outboundEnvelope struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type ClientMessageSend struct {
	MsgID          string `json:"msg_id"`
	ConversationID uint64 `json:"conversation_id"`
	Content        string `json:"content"`
}

type ClientMessageDelivered struct {
	ConversationID uint64 `json:"conversation_id"`
	DeliveredSeq   uint64 `json:"delivered_seq"`
}

type ClientMessageRead struct {
	ConversationID uint64 `json:"conversation_id"`
	ReadSeq        uint64 `json:"read_seq"`
}

type ServerMessage struct {
	ID             uint64             `json:"id"`
	MsgID          string             `json:"msg_id"`
	ConversationID uint64             `json:"conversation_id"`
	Seq            uint64             `json:"seq"`
	Type           model.MessageType  `json:"type"`
	Event          model.MessageEvent `json:"event"`
	From           uint64             `json:"from"`
	SendTime       int64              `json:"send_time"`
	Content        string             `json:"content"`
	Extra          json.RawMessage    `json:"extra,omitempty"`
}

type SyncRequiredData struct {
	ConversationID uint64 `json:"conversation_id,omitempty"`
	Reason         string `json:"reason"`
}

type MessageDeliveredData struct {
	ConversationID uint64 `json:"conversation_id"`
	UserID         uint64 `json:"user_id"`
	DeliveredSeq   uint64 `json:"delivered_seq"`
}

type MessageReadData struct {
	ConversationID uint64 `json:"conversation_id"`
	UserID         uint64 `json:"user_id"`
	ReadSeq        uint64 `json:"read_seq"`
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
		Type: eventType,
		Data: data,
	})
}

func DecodeEnvelope(payload []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return Envelope{}, apperr.MessageInvalidPayload()
	}
	if env.Type == "" || len(env.Data) == 0 {
		return Envelope{}, apperr.MessageInvalidPayload()
	}
	return env, nil
}

func DecodeClientMessageSendData(payload []byte) (ClientMessageSend, error) {
	var req ClientMessageSend
	if err := json.Unmarshal(payload, &req); err != nil {
		return ClientMessageSend{}, apperr.MessageInvalidPayload()
	}
	return req, nil
}

func DecodeClientMessageDeliveredData(payload []byte) (ClientMessageDelivered, error) {
	var req ClientMessageDelivered
	if err := json.Unmarshal(payload, &req); err != nil {
		return ClientMessageDelivered{}, apperr.MessageInvalidPayload()
	}
	return req, nil
}

func DecodeClientMessageReadData(payload []byte) (ClientMessageRead, error) {
	var req ClientMessageRead
	if err := json.Unmarshal(payload, &req); err != nil {
		return ClientMessageRead{}, apperr.MessageInvalidPayload()
	}
	return req, nil
}

func NewServerMessage(msg model.Message, publicIDs map[uint64]uint64) ServerMessage {
	return ServerMessage{
		ID:             msg.ID,
		MsgID:          msg.MsgID,
		ConversationID: msg.ConversationID,
		Seq:            msg.Seq,
		Type:           msg.Type,
		Event:          msg.Event,
		From:           publicIDs[msg.From],
		SendTime:       msg.SendTime,
		Content:        msg.Content,
		Extra:          msg.Extra,
	}
}
