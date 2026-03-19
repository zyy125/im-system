package mq

import(
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (r *RabbitMQ) PublishChatMsg(ctx context.Context, msgJSON []byte) error {
	return r.ch.PublishWithContext(
		ctx,
		"",
		ChatQueueName,
		false,
		false,
		amqp.Publishing{
			ContentType:     "application/json",
			ContentEncoding: "utf-8",
			Body:            msgJSON,
		},
	)
}