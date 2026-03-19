package mq

import (
	"context"
	"encoding/json"
	"log"

	"github.com/zyy125/im-system/internal/model"
	amqp "github.com/rabbitmq/amqp091-go"
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

			var chatMsg model.ChatMsg
			if err := json.Unmarshal(d.Body, &chatMsg); err != nil {
				log.Printf("consume chat msg unmarshal error: %v", err)
				_ = d.Nack(false, false)
				continue
			}

			if err := r.msgSvc.SaveMsg(ctx, &chatMsg); err != nil {
				log.Printf("consume chat msg save error: %v", err)
				_ = d.Nack(false, false)
				continue
			}

			if err := d.Ack(false); err != nil {
				log.Printf("consume chat msg ack error: %v", err)
			}
		}
	}
}