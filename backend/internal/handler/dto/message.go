package dto

import (
	"encoding/json"

	"github.com/zyy125/im-system/internal/model"
)

type MessageResp struct {
	ID             uint64             `json:"id"`
	MsgID          string             `json:"msg_id"`
	ConversationID uint64             `json:"conversation_id"`
	Seq            uint64             `json:"seq"`
	Type           model.MessageType  `json:"type"`
	Event          model.MessageEvent `json:"event"`
	From           uint64             `json:"from"`
	SendTime       int64              `json:"send_time"`
	Content        string             `json:"content"`
	Extra          json.RawMessage    `json:"extra,omitempty" swaggertype:"object"`
}

type MessageHistoryResp struct {
	Messages      []MessageResp `json:"messages"`
	HasMore       bool          `json:"has_more"`
	NextBeforeSeq uint64        `json:"next_before_seq,omitempty"`
}

type MessageSyncResp struct {
	Messages       []MessageResp `json:"messages"`
	HasMore        bool          `json:"has_more"`
	MaxReturnedSeq uint64        `json:"max_returned_seq"`
}

type MarkReadReq struct {
	ConversationID uint64 `json:"conversation_id" binding:"required"`
	ReadSeq        uint64 `json:"read_seq" binding:"required"`
}
