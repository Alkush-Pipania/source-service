package docs

import (
	"context"

	"github.com/Alkush-Pipania/carter-go/pkg/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// Repository defines the interface for document-related DB operations
type Repository interface {
	// SaveContent stores the extracted text from the PDF/PPT
	SaveContent(ctx context.Context, sourceID pgtype.UUID, content string) error
	
	// UpdateStatus updates the processing status (e.g., 'processing', 'indexed', 'failed')
	UpdateStatus(ctx context.Context, sourceID pgtype.UUID, status db.SourceStatus) error
}

type repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) Repository {
	return &repository{q: q}
}

// SaveContent calls the CreateSourceContent SQL query
func (r *repository) SaveContent(ctx context.Context, sourceID pgtype.UUID, content string) error {
	// Note: We are using the same table 'source_contents' as links
	return r.q.CreateSourceContent(ctx, db.CreateSourceContentParams{
		SourceID:    sourceID,
		ContentText: content,
	})
}

// UpdateStatus calls the UpdateSourceStatus SQL query
func (r *repository) UpdateStatus(ctx context.Context, sourceID pgtype.UUID, status db.SourceStatus) error {
	return r.q.UpdateSourceStatus(ctx, db.UpdateSourceStatusParams{
		ID:     sourceID,
		Status: status,
	})
}