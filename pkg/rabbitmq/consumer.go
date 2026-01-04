package rabbitmq

import (
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	ch       *amqp091.Channel
	exchange string
	queue    string
}

type ConsumerConfig struct {
	Exchange     string
	ExchangeType string
	Queue        string
	RoutingKey   string
}

func NewChannel(conn *amqp091.Connection, cfg ConsumerConfig) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	err = ch.ExchangeDeclare(
		cfg.Exchange,
		cfg.ExchangeType,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// create queue
	q, err := ch.QueueDeclare(
		cfg.Queue, // Name
		true,      // Durable
		false,     // Delete when unused
		false,     // Exclusive
		false,     // No-wait
		nil,       // Arguments
	)
	if err != nil {
		return nil, err
	}

	// bind it
	err = ch.QueueBind(
		q.Name,
		cfg.RoutingKey, // Routing Key
		cfg.Exchange,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		ch:       ch,
		exchange: cfg.Exchange,
		queue:    q.Name,
	}, nil
}

func (p *Consumer) Close() error {
	return p.ch.Close()
}

func (c *Consumer) Start(handler func(amqp091.Delivery)) error {
	if err := c.ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	); err != nil {
		return err
	}

	// 2. Start Consuming
	msgs, err := c.ch.Consume(
		c.queue, // Queue name
		"",      // Consumer tag (empty = auto-generated)
		true,    // Auto-Ack: TRUE (User requested auto-ack)
		false,   // Exclusive
		false,   // No-local
		false,   // No-wait
		nil,     // Args
	)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			// Pass the message to the handler function
			// We process sequentially because prefetch count is 1
			handler(d)
		}
		fmt.Println("Channel closed, stopping consumer")
	}()

	return nil
}
