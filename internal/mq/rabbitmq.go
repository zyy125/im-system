package mq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/zyy125/im-system/internal/service"

)

type RabbitMQ struct {
	conn *amqp.Connection
	ch *amqp.Channel

	msgSvc *service.MessageService
}

const ChatQueueName = "chat_msg_queue"

func NewRabbitMQ(url string, msgSvc *service.MessageService) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	_, err = ch.QueueDeclare(
		ChatQueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	log.Println("RabbitMQ initialized")

	return &RabbitMQ{conn: conn, ch: ch, msgSvc: msgSvc}, nil
}