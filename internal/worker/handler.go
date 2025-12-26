// internal/worker/handler.go
package worker

import (
	"log"

	"github.com/rabbitmq/amqp091-go"
)

func HandleMessage(msg amqp091.Delivery) {
	log.Printf("Received: %s", msg.Body)

	// process message
	err := process(msg.Body)
	if err != nil {
		msg.Nack(false, true) // retry
		return
	}

	msg.Ack(false)
}

func process(data []byte) error {
	// your business logic
	return nil
}
