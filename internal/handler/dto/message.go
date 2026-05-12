package dto

import "github.com/zyy125/im-system/internal/model"

type MessageHistoryResp struct {
	Messages      []model.Message `json:"messages"`
	HasMore       bool            `json:"has_more"`
	NextBeforeSeq uint64          `json:"next_before_seq,omitempty"`
}

type MessageSyncResp struct {
	Messages       []model.Message `json:"messages"`
	HasMore        bool            `json:"has_more"`
	MaxReturnedSeq uint64          `json:"max_returned_seq"`
}

type MarkReadReq struct {
	ConversationID uint64 `json:"conversation_id" binding:"required"`
	ReadSeq        uint64 `json:"read_seq" binding:"required"`
}
