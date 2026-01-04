package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// RabbitMQ
	RabbitMQUrl  string
	Exchange     string
	QueueName    string
	ExchangeType string
	RoutingKey   string

	// Database
	DbUrl string
	Env   string

	// DigitalOcean Spaces (S3-compatible)
	DORegion    string
	DOEndpoint  string
	DOAccessKey string
	DOSecretKey string
	DOBucket    string

	// Gemini
	GeminiAPIKey string

	// Pinecone
	PineconeAPIKey string
	PineconeHost   string

	// LlamaParse
	LlamaParseAPIKey       string
	LlamaParsePollInterval time.Duration
	LlamaParseMaxRetries   int
}

func LoadEnv() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file, using environment variables")
	}
	return &Config{
		// RabbitMQ
		RabbitMQUrl:  getkey("RABBITMQURL", "amqp://guest:guest@localhost:5672/"),
		Exchange:     getkey("EXCHANGE", "carter.embedding"),
		QueueName:    getkey("QUEUENAME", "source.processor.queue"),
		ExchangeType: getkey("EXCHANGETYPE", "direct"),
		RoutingKey:   getkey("ROUTINGKEY", "source.process"),

		// Database
		DbUrl: getkey("DB_URL", ""),
		Env:   getkey("ENV", "development"),

		// DigitalOcean Spaces
		DORegion:    getkey("DO_REGION", ""),
		DOEndpoint:  getkey("DO_ENDPOINT", ""),
		DOAccessKey: getkey("DO_ACCESS_KEY", ""),
		DOSecretKey: getkey("DO_SECRET_KEY", ""),
		DOBucket:    getkey("DO_BUCKET", ""),

		// Gemini
		GeminiAPIKey: getkey("GEMINI_API_KEY", ""),

		// Pinecone
		PineconeAPIKey: getkey("PINECONE_API_KEY", ""),
		PineconeHost:   getkey("PINECONE_HOST", ""),

		// LlamaParse
		LlamaParseAPIKey:       getkey("LLAMAPARSE_API_KEY", ""),
		LlamaParsePollInterval: time.Duration(getEnvValue(os.Getenv("LLAMAPARSE_POLL_INTERVAL_SECONDS"), 2)) * time.Second,
		LlamaParseMaxRetries:   getEnvValue(os.Getenv("LLAMAPARSE_MAX_RETRIES"), 150),
	}
}

func getkey(key string, fallback string) string {
	if os.Getenv(key) == "" {
		return fallback
	}
	return os.Getenv(key)
}

func getEnvValue(s string, fallback int) int {
	if s == "" {
		return fallback
	}

	value, err := strconv.Atoi(s)

	if err != nil {
		return fallback
	}
	return value
}
