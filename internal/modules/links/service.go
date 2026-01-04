package links

import (
	"context"
	"fmt"
	"log"

	"github.com/Alkush-Pipania/source-service/internal/modules"
	"github.com/Alkush-Pipania/source-service/pkg/client/gemini"
	"github.com/Alkush-Pipania/source-service/pkg/client/pinecone"
	"github.com/Alkush-Pipania/source-service/pkg/client/s3"
	"github.com/Alkush-Pipania/source-service/pkg/db"
	"github.com/Alkush-Pipania/source-service/pkg/utils"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	// Key prefix for storing link images in S3
	ImageKeyPrefix = "links"
)

type Service struct {
	repo      Repository
	processor *LinkProcessor
	gemini    *gemini.Client
	pinecone  *pinecone.Client
	s3        *s3.Client
}

// NewService creates a new links service
func NewService(repo Repository, proc *LinkProcessor, gem *gemini.Client, pine *pinecone.Client, s3Client *s3.Client) *Service {
	return &Service{
		repo:      repo,
		processor: proc,
		gemini:    gem,
		pinecone:  pine,
		s3:        s3Client,
	}
}

func (s *Service) ProcessLink(ctx context.Context, job modules.SourceJob) error {
	log.Printf("Starting processing for link: %s", job.OriginalURL)

	var sourceUUID pgtype.UUID
	if err := sourceUUID.Scan(job.SourceID); err != nil {
		return fmt.Errorf("invalid source id: %w", err)
	}

	// 1. Scrape Content
	content, err := s.processor.Process(ctx, job)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, sourceUUID, db.SourceStatusFailed)
		return err
	}

	// 2. Upload image to S3 if available
	var imageS3URL string
	if imgURL, ok := content.Metadata["image_url"].(string); ok && imgURL != "" {
		keyPrefix := fmt.Sprintf("%s/%s", ImageKeyPrefix, job.UserID)
		s3URL, err := s.s3.UploadFromURL(ctx, imgURL, keyPrefix)
		if err != nil {
			log.Printf("Warning: Failed to upload image to S3: %v", err)
			// Continue without image, don't fail the whole process
		} else {
			imageS3URL = s3URL
			log.Printf("Image uploaded to S3: %s", s3URL)
		}
	}

	// 3. Update title and image in DB
	if err := s.repo.UpdateTitleAndImage(ctx, sourceUUID, content.Title, imageS3URL); err != nil {
		log.Printf("Warning: Failed to update title/image: %v", err)
	}

	// 4. Save raw text to DB (Skipped as per user request to remove source content)
	// if err := s.repo.SaveContent(ctx, sourceUUID, content.Text); err != nil {
	// 	_ = s.repo.UpdateStatus(ctx, sourceUUID, db.SourceStatusFailed)
	// 	return err
	// }

	// 5. Chunking (1000 chars per chunk, 200 overlap)
	chunks := utils.SplitText(content.Text, 1000, 200)
	var vectors []pinecone.Vector

	// 6. Generate Embeddings & Prepare Vectors
	for _, chunk := range chunks {
		embedding, err := s.gemini.GenerateEmbedding(ctx, chunk.Text)
		if err != nil {
			log.Printf("Failed to embed chunk %d: %v", chunk.Index, err)
			continue
		}

		// Create Vector ID: sourceID_chunkIndex
		vectorID := fmt.Sprintf("%s_%d", job.SourceID, chunk.Index)

		vectors = append(vectors, pinecone.Vector{
			ID:     vectorID,
			Values: embedding,
			Metadata: map[string]interface{}{
				"source_id":   job.SourceID,
				"text":        chunk.Text,
				"url":         job.OriginalURL,
				"title":       content.Title,
				"chunk_index": chunk.Index,
				"type":        "link",
			},
		})
	}

	// 7. Upsert to Pinecone with userID as namespace
	if len(vectors) > 0 {
		if _, err := s.pinecone.UpsertWithNamespace(ctx, job.UserID, vectors); err != nil {
			log.Printf("Failed to upsert vectors: %v", err)
			_ = s.repo.UpdateStatus(ctx, sourceUUID, db.SourceStatusFailed)
			return err
		}
	}

	// 8. Mark as Indexed
	if err := s.repo.UpdateStatus(ctx, sourceUUID, db.SourceStatusIndexed); err != nil {
		return err
	}

	log.Printf("Successfully processed and indexed link: %s", job.SourceID)
	return nil
}
