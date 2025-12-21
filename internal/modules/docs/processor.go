package docs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Alkush-Pipania/source-service/internal/modules"
	"github.com/Alkush-Pipania/source-service/pkg/client/lamaparse"
	"github.com/Alkush-Pipania/source-service/pkg/client/s3"
)

type DocProcessor struct {
	s3        *s3.Client
	lamaparse *lamaparse.Client
}

func NewDocProcessor(s3 *s3.Client, lp *lamaparse.Client) *DocProcessor {
	return &DocProcessor{
		s3:        s3,
		lamaparse: lp,
	}
}

func (p *DocProcessor) Process(ctx context.Context, job modules.SourceJob) (*modules.ProcessedContent, error) {
	if job.S3Bucket == "" || job.S3Key == "" {
		return nil, fmt.Errorf("missing s3 bucket or key")
	}

	// 1. Download file from S3 to Temp
	tempPath, err := p.s3.DownloadToTemp(ctx, job.S3Bucket, job.S3Key)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tempPath) // Cleanup temp file when done

	// 2. Determine File Type
	ext := strings.ToLower(filepath.Ext(job.S3Key))
	var contentText string

	// 3. Process based on type
	switch ext {
	case ".pdf", ".ppt", ".pptx", ".doc", ".docx":
		// Use LamaParse for complex docs
		if p.lamaparse == nil {
			return nil, fmt.Errorf("lamaparse client not configured")
		}
		contentText, err = p.lamaparse.ParseFile(ctx, tempPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse doc: %w", err)
		}

	case ".txt", ".md", ".csv":
		// Read plain text files directly
		bytes, err := os.ReadFile(tempPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read local file: %w", err)
		}
		contentText = string(bytes)

	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}

	// 4. Return result
	return &modules.ProcessedContent{
		Title: filepath.Base(job.S3Key), // Simple title, can be improved
		Text:  contentText,
		Metadata: map[string]interface{}{
			"s3_key":    job.S3Key,
			"s3_bucket": job.S3Bucket,
			"file_type": ext,
			"source":    "s3_document",
		},
	}, nil
}
