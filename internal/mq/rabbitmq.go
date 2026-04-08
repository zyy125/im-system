package mq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/zyy125/im-system/internal/service"
)

type RabbitMQ struct {
	conn *amqp.Connection
	ch   *amqp.Channel

	msgSvc service.MessageService
}

const ChatQueueName = "chat_msg_queue"
const ChatDeadLetterQueueName = "chat_msg_dead_letter_queue"

func NewRabbitMQ(url string, msgSvc service.MessageService) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := declareQueue(ch, ChatQueueName); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	if err := declareQueue(ch, ChatDeadLetterQueueName); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	log.Println("RabbitMQ initialized")

	return &RabbitMQ{conn: conn, ch: ch, msgSvc: msgSvc}, nil
}

func (r *RabbitMQ) Close() error {
	if r == nil {
		return nil
	}
	if r.ch != nil {
		_ = r.ch.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

func declareQueue(ch *amqp.Channel, name string) error {
	_, err := ch.QueueDeclare(
		name,
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}
