// pkg/rabbitmq/channel.go
package rabbitmq

import "github.com/rabbitmq/amqp091-go"

type Channel struct {
	Ch *amqp091.Channel
}

func NewChannel(conn *Connection) (*Channel, error) {
	ch, err := conn.Conn.Channel()
	if err != nil {
		return nil, err
	}
	return &Channel{Ch: ch}, nil
}
