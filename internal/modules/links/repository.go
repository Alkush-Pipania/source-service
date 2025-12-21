package links

import (
	"context"

	"github.com/Alkush-Pipania/source-service/pkg/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository interface {
	SaveContent(ctx context.Context, sourceID pgtype.UUID, content string) error
	UpdateStatus(ctx context.Context, sourceID pgtype.UUID, status db.SourceStatus) error
}

type repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) Repository {
	return &repository{q: q}
}

func (r *repository) SaveContent(ctx context.Context, sourceID pgtype.UUID, content string) error {
	return r.q.CreateSourceContent(ctx, db.CreateSourceContentParams{
		SourceID:    sourceID,
		ContentText: content,
	})
}

func (r *repository) UpdateStatus(ctx context.Context, sourceID pgtype.UUID, status db.SourceStatus) error {
	return r.q.UpdateSourceStatus(ctx, db.UpdateSourceStatusParams{
		ID:     sourceID,
		Status: status,
	})
}
