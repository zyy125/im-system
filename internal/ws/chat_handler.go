package ws

import (
	"context"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/service"
)

type ChatSendHandler interface {
	// HandleMessageSend 处理一条客户端发送的会话消息，并展开成需要投递给各成员的转发任务。
	HandleMessageSend(ctx context.Context, senderID uint64, req ClientMessageSend) ([]*ForwardMessage, error)
}

type MessageAckHandler interface {
	HandleMessageDelivered(ctx context.Context, userID uint64, req ClientMessageDelivered) ([]*ForwardMessage, error)
	HandleMessageRead(ctx context.Context, userID uint64, req ClientMessageRead) ([]*ForwardMessage, error)
}

type chatSendHandler struct {
	messageSendService service.MessageSendService
}

type messageAckHandler struct {
	messageService      service.MessageService
	conversationService service.ConversationService
}

func NewChatSendHandler(messageSendService service.MessageSendService) ChatSendHandler {
	return &chatSendHandler{messageSendService: messageSendService}
}

func NewMessageAckHandler(messageService service.MessageService, conversationService service.ConversationService) MessageAckHandler {
	return &messageAckHandler{
		messageService:      messageService,
		conversationService: conversationService,
	}
}

func (h *chatSendHandler) HandleMessageSend(ctx context.Context, senderID uint64, req ClientMessageSend) ([]*ForwardMessage, error) {
	if req.MsgID == "" || req.ConversationID == 0 || req.Content == "" {
		return nil, apperr.MessageInvalidPayload()
	}
	if h.messageSendService == nil {
		return nil, apperr.Internal("message send service unavailable", nil)
	}

	saved, recipients, err := h.messageSendService.SendTextMessage(ctx, senderID, req.ConversationID, req.MsgID, req.Content)
	if err != nil {
		return nil, err
	}

	sentPayload, err := MarshalEnvelope(EventTypeMessageSent, NewServerMessage(saved))
	if err != nil {
		return nil, err
	}
	createdPayload, err := MarshalEnvelope(EventTypeMessageCreated, NewServerMessage(saved))
	if err != nil {
		return nil, err
	}

	forwards := make([]*ForwardMessage, 0, len(recipients)+1)
	forwards = append(forwards, &ForwardMessage{
		To:      senderID,
		Content: sentPayload,
	})
	for _, recipientID := range recipients {
		if recipientID == senderID {
			continue
		}
		forwards = append(forwards, &ForwardMessage{
			To:      recipientID,
			Content: createdPayload,
		})
	}
	return forwards, nil
}

func (h *messageAckHandler) HandleMessageDelivered(ctx context.Context, userID uint64, req ClientMessageDelivered) ([]*ForwardMessage, error) {
	if req.ConversationID == 0 || req.DeliveredSeq == 0 {
		return nil, apperr.MessageInvalidPayload()
	}
	if h.messageService == nil {
		return nil, apperr.Internal("message service unavailable", nil)
	}
	recipients, err := h.messageService.MarkDelivered(ctx, userID, req.ConversationID, req.DeliveredSeq)
	if err != nil {
		return nil, err
	}
	payload, err := MarshalEnvelope(EventTypeMessageDelivered, MessageDeliveredData{
		ConversationID: req.ConversationID,
		UserID:         userID,
		DeliveredSeq:   req.DeliveredSeq,
	})
	if err != nil {
		return nil, err
	}
	return buildReceiptForwards(recipients, payload), nil
}

func (h *messageAckHandler) HandleMessageRead(ctx context.Context, userID uint64, req ClientMessageRead) ([]*ForwardMessage, error) {
	if req.ConversationID == 0 || req.ReadSeq == 0 {
		return nil, apperr.MessageInvalidPayload()
	}
	if h.conversationService == nil {
		return nil, apperr.Internal("conversation service unavailable", nil)
	}
	recipients, err := h.conversationService.MarkRead(ctx, userID, req.ConversationID, req.ReadSeq)
	if err != nil {
		return nil, err
	}
	payload, err := MarshalEnvelope(EventTypeMessageRead, MessageReadData{
		ConversationID: req.ConversationID,
		UserID:         userID,
		ReadSeq:        req.ReadSeq,
	})
	if err != nil {
		return nil, err
	}
	return buildReceiptForwards(recipients, payload), nil
}

func buildReceiptForwards(recipients []uint64, payload []byte) []*ForwardMessage {
	forwards := make([]*ForwardMessage, 0, len(recipients))
	for _, recipientID := range recipients {
		forwards = append(forwards, &ForwardMessage{
			To:      recipientID,
			Content: payload,
		})
	}
	return forwards
}
