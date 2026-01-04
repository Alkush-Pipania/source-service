package app

import (
	"context"

	"github.com/Alkush-Pipania/source-service/config"
	"github.com/Alkush-Pipania/source-service/internal/modules/docs"
	"github.com/Alkush-Pipania/source-service/internal/modules/links"
	"github.com/Alkush-Pipania/source-service/internal/modules/notes"
	"github.com/Alkush-Pipania/source-service/pkg/client/gemini"
	"github.com/Alkush-Pipania/source-service/pkg/client/lamaparse"
	"github.com/Alkush-Pipania/source-service/pkg/client/pinecone"
	"github.com/Alkush-Pipania/source-service/pkg/client/s3"
	"github.com/Alkush-Pipania/source-service/pkg/db"
)

// Clients holds all external API clients
type Clients struct {
	Gemini     *gemini.Client
	Pinecone   *pinecone.Client
	LlamaParse *lamaparse.Client
	S3         *s3.Client
}

// Services holds all module services
type Services struct {
	Links *links.Service
	Notes *notes.Service
	Docs  *docs.Service
}

type Container struct {
	DB       *db.Queries
	Clients  *Clients
	Services *Services
}

func NewContainer(ctx context.Context, cfg *config.Config, queries *db.Queries, clients *Clients) (*Container, func(), error) {
	// Initialize repositories
	linksRepo := links.NewRepository(queries)
	notesRepo := notes.NewRepository(queries)
	docsRepo := docs.NewRepository(queries)

	// Initialize processors
	linkProcessor := links.NewLinkProcessor()
	docProcessor := docs.NewDocProcessor(clients.S3, clients.LlamaParse)

	// Initialize services
	linksService := links.NewService(linksRepo, linkProcessor, clients.Gemini, clients.Pinecone, clients.S3)
	notesService := notes.NewService(notesRepo, clients.Gemini, clients.Pinecone)
	docsService := docs.NewService(docsRepo, docProcessor)

	services := &Services{
		Links: linksService,
		Notes: notesService,
		Docs:  docsService,
	}

	return &Container{
		DB:       queries,
		Clients:  clients,
		Services: services,
	}, func() {}, nil
}
