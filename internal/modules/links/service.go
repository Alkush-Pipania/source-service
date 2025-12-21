package links

import (
	"context"
	"fmt"
	"log"

	"github.com/Alkush-Pipania/source-service/internal/modules"
	"github.com/Alkush-Pipania/source-service/pkg/client/gemini"
	"github.com/Alkush-Pipania/source-service/pkg/client/pinecone"
	"github.com/Alkush-Pipania/source-service/pkg/db"
	"github.com/Alkush-Pipania/source-service/pkg/utils"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	repo      Repository
	processor *LinkProcessor
	gemini    *gemini.Client
	pinecone  *pinecone.Client
}

// Update constructor to accept new clients
func NewService(repo Repository, proc *LinkProcessor, gem *gemini.Client, pine *pinecone.Client) *Service {
	return &Service{
		repo:      repo,
		processor: proc,
		gemini:    gem,
		pinecone:  pine,
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

	// 2. Save raw text to DB (for backup/display)
	if err := s.repo.SaveContent(ctx, sourceUUID, content.Text); err != nil {
		_ = s.repo.UpdateStatus(ctx, sourceUUID, db.SourceStatusFailed)
		return err
	}

	// 3. Chunking
	// 1000 chars per chunk, 200 overlap
	chunks := utils.SplitText(content.Text, 1000, 200)
	var vectors []pinecone.Vector

	// 4. Generate Embeddings & Prepare Vectors
	for _, chunk := range chunks {
		embedding, err := s.gemini.GenerateEmbedding(ctx, chunk.Text)
		if err != nil {
			log.Printf("Failed to embed chunk %d: %v", chunk.Index, err)
			continue // Skip bad chunks or fail? strict: fail.
		}

		// Create Vector ID: sourceID_chunkIndex
		vectorID := fmt.Sprintf("%s_%d", job.SourceID, chunk.Index)

		vectors = append(vectors, pinecone.Vector{
			ID:     vectorID,
			Values: embedding,
			Metadata: map[string]interface{}{
				"source_id":   job.SourceID,
				"text":        chunk.Text, // Storing text in metadata for RAG retrieval
				"url":         job.OriginalURL,
				"title":       content.Title,
				"chunk_index": chunk.Index,
				"type":        "link",
			},
		})
	}

	// 5. Upsert to Pinecone (Batching could be added here for huge files)
	if len(vectors) > 0 {
		if err := s.pinecone.Upsert(ctx, vectors); err != nil {
			log.Printf("Failed to upsert vectors: %v", err)
			_ = s.repo.UpdateStatus(ctx, sourceUUID, db.SourceStatusFailed)
			return err
		}
	}

	// 6. Mark as Indexed
	if err := s.repo.UpdateStatus(ctx, sourceUUID, db.SourceStatusIndexed); err != nil {
		return err
	}

	log.Printf("Successfully processed and indexed link: %s", job.SourceID)
	return nil
}
