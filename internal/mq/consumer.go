package mq

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
)

var workerNum = 1

func (r *RabbitMQ) ConsumeChatMsg(ctx context.Context) error {
	if err := r.ch.Qos(workerNum, 0, false); err != nil {
		return err
	}

	msgs, err := r.ch.Consume(
		ChatQueueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for i := 0; i < workerNum; i++ {
		go r.worker(ctx, msgs)
	}

	return nil
}

func (r *RabbitMQ) worker(ctx context.Context, msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return

		case d, ok := <-msgs:
			if !ok {
				return
			}

			var chatMsg model.ChatMessage
			if err := json.Unmarshal(d.Body, &chatMsg); err != nil {
				log.Printf("consume chat msg unmarshal error: %v", err)
				r.handleDeadLetter(ctx, d, "invalid_json", err)
				continue
			}

			if _, err := r.msgSvc.SaveMessage(ctx, &chatMsg); err != nil {
				log.Printf(
					"consume chat msg save error: msg_id=%s conversation_id=%s from=%d to=%d err=%v",
					chatMsg.MsgID, chatMsg.ConversationID, chatMsg.From, chatMsg.To, err,
				)

				if isPermanentChatMsgError(err) {
					r.handleDeadLetter(ctx, d, "permanent_save_error", err)
				} else {
					_ = d.Nack(false, true)
				}
				continue
			}

			if err := d.Ack(false); err != nil {
				log.Printf("consume chat msg ack error: %v", err)
			}
		}
	}
}

func (r *RabbitMQ) handleDeadLetter(ctx context.Context, d amqp.Delivery, reason string, cause error) {
	if err := r.publishDeadLetter(ctx, d, reason, cause); err != nil {
		log.Printf("publish dead letter failed: reason=%s err=%v", reason, err)
		_ = d.Nack(false, true)
		return
	}
	if err := d.Ack(false); err != nil {
		log.Printf("ack dead letter source msg failed: reason=%s err=%v", reason, err)
	}
}

func (r *RabbitMQ) publishDeadLetter(ctx context.Context, d amqp.Delivery, reason string, cause error) error {
	headers := amqp.Table{
		"reason":               reason,
		"error":                cause.Error(),
		"original_routing_key": d.RoutingKey,
	}

	return r.ch.PublishWithContext(
		ctx,
		"",
		ChatDeadLetterQueueName,
		false,
		false,
		amqp.Publishing{
			ContentType:     d.ContentType,
			ContentEncoding: d.ContentEncoding,
			Body:            d.Body,
			Headers:         headers,
		},
	)
}

func isPermanentChatMsgError(err error) bool {
	if err == nil {
		return false
	}

	switch apperr.CodeOf(err) {
	case apperr.CodeInvalidArgument,
		apperr.CodeMessageIDRequired,
		apperr.CodeMessageConversationRequired,
		apperr.CodeConversationMemberNotFound,
		apperr.CodeConversationInvalidSingleKey,
		apperr.CodeFriendCannotAddSelf:
		return true
	}

	var numErr *strconv.NumError
	if errors.As(err, &numErr) {
		return true
	}
	return false
}
