package docs

import (
	"context"
	"fmt"
	"log"

	"github.com/Alkush-Pipania/source-service/internal/modules"
	"github.com/Alkush-Pipania/source-service/pkg/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	repo      Repository
	processor *DocProcessor
}

func NewService(repo Repository, proc *DocProcessor) *Service {
	return &Service{
		repo:      repo,
		processor: proc,
	}
}

func (s *Service) ProcessDoc(ctx context.Context, job modules.SourceJob) error {
	log.Printf("Processing document: %s/%s", job.S3Bucket, job.S3Key)

	var sourceUUID pgtype.UUID
	if err := sourceUUID.Scan(job.SourceID); err != nil {
		return fmt.Errorf("invalid source id: %w", err)
	}

	// 1. Download & Parse
	content, err := s.processor.Process(ctx, job)
	if err != nil {
		log.Printf("Doc processing failed: %v", err)
		_ = s.repo.UpdateStatus(ctx, sourceUUID, db.SourceStatusFailed)
		return err
	}

	// 2. Save Content
	if err := s.repo.SaveContent(ctx, sourceUUID, content.Text); err != nil {
		_ = s.repo.UpdateStatus(ctx, sourceUUID, db.SourceStatusFailed)
		return err
	}

	// 3. Mark as Indexed (Queue for embedding in future)
	if err := s.repo.UpdateStatus(ctx, sourceUUID, db.SourceStatusIndexed); err != nil {
		return err
	}

	log.Printf("Successfully processed document: %s", job.SourceID)
	return nil
}