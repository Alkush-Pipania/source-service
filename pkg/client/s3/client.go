package s3

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	client     *s3.Client
	downloader *manager.Downloader
}

func NewClient(ctx context.Context) (*Client, error) {
	// Automatically loads credentials from AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	return &Client{
		client:     client,
		downloader: manager.NewDownloader(client),
	}, nil
}

// DownloadToTemp downloads a file from S3 to a temporary file on disk
// Returns the path to the temp file. Caller is responsible for deleting it.
func (c *Client) DownloadToTemp(ctx context.Context, bucket, key string) (string, error) {
	// Create a temp file
	tempFile, err := os.CreateTemp("", "carter-source-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Download
	_, err = c.downloader.Download(ctx, tempFile, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Clean up if download failed
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to download file from s3: %w", err)
	}

	return tempFile.Name(), nil
}