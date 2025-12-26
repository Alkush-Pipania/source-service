// cmd/main.go
package main

import (
	"log"

	"github.com/Alkush-Pipania/source-service/internal/worker"
	"github.com/Alkush-Pipania/source-service/pkg/rabbitmq"
)

func main() {
	cfg := rabbitmq.Config{
		URL:          "amqp://guest:guest@localhost:5672/",
		Exchange:     "events.exchange",
		ExchangeType: "topic",
		Queue:        "events.consumer.email",
		RoutingKey:   "events.email.*",
		ConsumerTag:  "email-service",
	}

	conn, err := rabbitmq.NewRabbitClient(cfg.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := rabbitmq.NewChannel(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Ch.Close()

	consumer := worker.NewConsumer(ch.Ch, cfg)
	err = consumer.Start(worker.HandleMessage)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Consumer started")
	select {} // block forever
}
