package notes

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
	repo     Repository
	gemini   *gemini.Client
	pinecone *pinecone.Client
}

func NewService(repo Repository, gem *gemini.Client, pine *pinecone.Client) *Service {
	return &Service{
		repo:     repo,
		gemini:   gem,
		pinecone: pine,
	}
}

func (s *Service) ProcessNote(ctx context.Context, job modules.SourceJob) error {
	log.Printf("Processing note: %s", job.SourceID)

	var sourceUUID pgtype.UUID
	if err := sourceUUID.Scan(job.SourceID); err != nil {
		return fmt.Errorf("invalid source id: %w", err)
	}

	// 1. Fetch Content from DB
	// (API already saved it, we just need to read it to embed it)
	text, err := s.repo.GetContent(ctx, sourceUUID)
	if err != nil {
		log.Printf("Failed to get note content: %v", err)
		_ = s.repo.UpdateStatus(ctx, sourceUUID, db.SourceStatusFailed)
		return err
	}

	// 2. Chunking
	// Notes might be short, but we still chunk to be safe and consistent
	chunks := utils.SplitText(text, 1000, 200)
	var vectors []pinecone.Vector

	// 3. Generate Embeddings
	for _, chunk := range chunks {
		embedding, err := s.gemini.GenerateEmbedding(ctx, chunk.Text)
		if err != nil {
			log.Printf("Failed to embed note chunk %d: %v", chunk.Index, err)
			continue
		}

		vectorID := fmt.Sprintf("%s_%d", job.SourceID, chunk.Index)

		vectors = append(vectors, pinecone.Vector{
			ID:     vectorID,
			Values: embedding,
			Metadata: map[string]interface{}{
				"source_id":   job.SourceID,
				"text":        chunk.Text,
				"title":       "Note", // Notes usually don't have titles in the content, maybe pass from job?
				"chunk_index": chunk.Index,
				"type":        "note",
			},
		})
	}

	// 4. Upsert to Pinecone
	if len(vectors) > 0 {
		if err := s.pinecone.Upsert(ctx, vectors); err != nil {
			log.Printf("Failed to upsert note vectors: %v", err)
			_ = s.repo.UpdateStatus(ctx, sourceUUID, db.SourceStatusFailed)
			return err
		}
	}

	// 5. Mark as Indexed
	if err := s.repo.UpdateStatus(ctx, sourceUUID, db.SourceStatusIndexed); err != nil {
		return err
	}

	log.Printf("Successfully processed note: %s", job.SourceID)
	return nil
}
