package rabbitmq

import "github.com/rabbitmq/amqp091-go"

type RabbitClient struct {
	Conn *amqp091.Connection
}

func NewRabbitClient(url string) (*RabbitClient, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, err
	}
	return &RabbitClient{
		Conn: conn,
	}, nil
}

func (rc *RabbitClient) Close() error {
	if rc.Conn != nil {
		return rc.Conn.Close()
	}
	return nil
}
