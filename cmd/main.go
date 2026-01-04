// cmd/main.go
package main

import (
	"context"
	"log"

	"github.com/Alkush-Pipania/source-service/config"
	"github.com/Alkush-Pipania/source-service/internal/app"
	"github.com/Alkush-Pipania/source-service/internal/worker"
	"github.com/Alkush-Pipania/source-service/pkg/client/gemini"
	"github.com/Alkush-Pipania/source-service/pkg/client/lamaparse"
	"github.com/Alkush-Pipania/source-service/pkg/client/pinecone"
	"github.com/Alkush-Pipania/source-service/pkg/client/s3"
	"github.com/Alkush-Pipania/source-service/pkg/db"
	"github.com/Alkush-Pipania/source-service/pkg/rabbitmq"
)

func main() {
	cfg := config.LoadEnv()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database
	dbConn := db.Init(ctx, cfg.DbUrl)
	q := db.New(dbConn)

	// RabbitMQ
	conn, err := rabbitmq.NewRabbitClient(cfg.RabbitMQUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := rabbitmq.NewChannel(conn.Conn, rabbitmq.ConsumerConfig{
		Exchange:     cfg.Exchange,
		ExchangeType: cfg.ExchangeType,
		Queue:        cfg.QueueName,
		RoutingKey:   cfg.RoutingKey,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	// Initialize S3/DigitalOcean Spaces client
	s3Client, err := s3.NewClient(ctx, s3.ClientConfig{
		Region:     cfg.DORegion,
		Endpoint:   cfg.DOEndpoint,
		AccessKey:  cfg.DOAccessKey,
		SecretKey:  cfg.DOSecretKey,
		BucketName: cfg.DOBucket,
	})
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}
	log.Println("DigitalOcean Spaces client initialized")

	// Initialize Gemini client
	geminiClient, err := gemini.NewClient(ctx, cfg.GeminiAPIKey)
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}
	log.Println("Gemini client initialized")

	// Initialize Pinecone client
	pineconeClient, err := pinecone.NewClient(
		cfg.PineconeAPIKey,
		cfg.PineconeHost,
	)
	if err != nil {
		log.Fatalf("Failed to create Pinecone client: %v", err)
	}
	defer pineconeClient.Close()
	log.Println("Pinecone client initialized")

	// Initialize LlamaParse client
	llamaParseClient := lamaparse.NewClientWithConfig(lamaparse.ClientConfig{
		APIKey:         cfg.LlamaParseAPIKey,
		PollInterval:   cfg.LlamaParsePollInterval,
		MaxPollRetries: cfg.LlamaParseMaxRetries,
	})
	log.Println("LlamaParse client initialized")

	// Create clients bundle
	clients := &app.Clients{
		Gemini:     geminiClient,
		Pinecone:   pineconeClient,
		LlamaParse: llamaParseClient,
		S3:         s3Client,
	}

	// Initialize container with all services
	container, cleanup, err := app.NewContainer(ctx, cfg, q, clients)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}
	defer cleanup()

	// Create worker with services and db
	w := worker.NewWorker(container.Services, q)

	// Start consumer
	err = ch.Start(w.HandleMessage)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Consumer started")
	select {} // block forever
}
