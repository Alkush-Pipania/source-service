// internal/worker/handler.go
package worker

import (
	"context"
	"encoding/json"
	"log"

	"github.com/Alkush-Pipania/source-service/internal/app"
	"github.com/Alkush-Pipania/source-service/internal/modules"
	"github.com/Alkush-Pipania/source-service/pkg/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rabbitmq/amqp091-go"
)

type Worker struct {
	services *app.Services
	db       *db.Queries
}

func NewWorker(services *app.Services, queries *db.Queries) *Worker {
	return &Worker{
		services: services,
		db:       queries,
	}
}

func (w *Worker) HandleMessage(msg amqp091.Delivery) {
	log.Printf("Received message: %s", msg.Body)

	ctx := context.Background()

	// Parse message from queue
	var message modules.SourceProcessingMessage
	if err := json.Unmarshal(msg.Body, &message); err != nil {
		log.Printf("Failed to parse message: %v", err)
		return
	}

	// Convert source ID to UUID
	var sourceUUID pgtype.UUID
	if err := sourceUUID.Scan(message.SourceID); err != nil {
		log.Printf("Invalid source ID: %v", err)
		return
	}

	// Fetch full source details from DB
	source, err := w.db.GetSourceByID(ctx, sourceUUID)
	if err != nil {
		log.Printf("Failed to get source from DB: %v", err)
		return
	}

	// Build enriched job
	job := modules.SourceJob{
		SourceID:    message.SourceID,
		Type:        message.Type,
		UserID:      message.UserID,
		OriginalURL: source.OriginalUrl.String,
		S3Bucket:    source.S3Bucket.String,
		S3Key:       source.S3Key.String,
		Title:       source.Title,
	}

	// Process based on job type
	switch job.Type {
	case "link":
		err = w.services.Links.ProcessLink(ctx, job)
	case "note":
		err = w.services.Notes.ProcessNote(ctx, job)
	case "pdf", "ppt", "doc":
		// All document types go through docs processor
		err = w.services.Docs.ProcessDoc(ctx, job)
	default:
		log.Printf("Unknown job type: %s", job.Type)
		return
	}

	if err != nil {
		log.Printf("Failed to process %s job: %v", job.Type, err)
		return
	}

	log.Printf("Successfully processed %s job: %s", job.Type, job.SourceID)
}
