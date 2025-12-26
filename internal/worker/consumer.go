// internal/worker/consumer.go
package worker

import (
	"github.com/Alkush-Pipania/source-service/pkg/rabbitmq"
	"github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	ch  *amqp091.Channel
	cfg rabbitmq.Config
}

func NewConsumer(ch *amqp091.Channel, cfg rabbitmq.Config) *Consumer {
	return &Consumer{ch: ch, cfg: cfg}
}

func (c *Consumer) Start(handler func(amqp091.Delivery)) error {
	// Exchange
	err := c.ch.ExchangeDeclare(
		c.cfg.Exchange,
		c.cfg.ExchangeType,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Queue
	q, err := c.ch.QueueDeclare(
		c.cfg.Queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Bind
	err = c.ch.QueueBind(
		q.Name,
		c.cfg.RoutingKey,
		c.cfg.Exchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// QoS (IMPORTANT)
	err = c.ch.Qos(1, 0, false)
	if err != nil {
		return err
	}

	msgs, err := c.ch.Consume(
		q.Name,
		c.cfg.ConsumerTag,
		false, // autoAck = false (VERY IMPORTANT)
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			handler(msg)
		}
	}()

	return nil
}
