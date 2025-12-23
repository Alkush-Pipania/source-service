package notes

import (
	"context"
	"fmt"

	"github.com/Alkush-Pipania/source-service/pkg/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository interface {
	GetContent(ctx context.Context, sourceID pgtype.UUID) (string, error)
	UpdateStatus(ctx context.Context, sourceID pgtype.UUID, status db.SourceStatus) error
}

type repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) Repository {
	return &repository{q: q}
}

func (r *repository) GetContent(ctx context.Context, sourceID pgtype.UUID) (string, error) {
	// 1. Fetch from source_contents table
	// Note: The SQL generates 'GetSourceContentBySourceID' which returns a slice ([]db.SourceContent)
	contents, err := r.q.GetSourceContentBySourceID(ctx, sourceID)
	if err != nil {
		return "", err
	}
	if len(contents) == 0 {
		return "", fmt.Errorf("no content found for source id: %v", sourceID)
	}

	// Return the text of the most recent entry (or the only one)
	return contents[0].ContentText, nil
}

func (r *repository) UpdateStatus(ctx context.Context, sourceID pgtype.UUID, status db.SourceStatus) error {
	return r.q.UpdateSourceStatus(ctx, db.UpdateSourceStatusParams{
		ID:     sourceID,
		Status: status,
	})
}
