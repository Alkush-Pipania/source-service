package links

import (
	"context"
	"fmt"
	"time"

	"github.com/Alkush-Pipania/source-service/internal/modules"
	"github.com/go-shiori/go-readability"
)

type LinkProcessor struct{}

func NewLinkProcessor() *LinkProcessor {
	return &LinkProcessor{}
}

// Process visits the URL and extracts the main article text and image
func (l *LinkProcessor) Process(ctx context.Context, job modules.SourceJob) (*modules.ProcessedContent, error) {
	if job.OriginalURL == "" {
		return nil, fmt.Errorf("original URL is missing")
	}

	// 1. Scrape with 30s timeout
	article, err := readability.FromURL(job.OriginalURL, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape url: %w", err)
	}

	// 2. Return clean text with image URL
	return &modules.ProcessedContent{
		Title: article.Title,
		Text:  article.TextContent,
		Metadata: map[string]interface{}{
			"original_url": job.OriginalURL,
			"site_name":    article.SiteName,
			"image_url":    article.Image, // Featured image from the page
			"favicon":      article.Favicon,
		},
	}, nil
}
