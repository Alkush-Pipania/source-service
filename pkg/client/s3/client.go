package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	s3Client   *s3.Client
	downloader *manager.Downloader
	uploader   *manager.Uploader
	bucketName string
	endpoint   string
}

type ClientConfig struct {
	Region     string
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
}

func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKey,
				cfg.SecretKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = false // Use virtual-hosted style (bucket.endpoint) for DigitalOcean Spaces
	})

	return &Client{
		s3Client:   s3Client,
		downloader: manager.NewDownloader(s3Client),
		uploader:   manager.NewUploader(s3Client),
		bucketName: cfg.BucketName,
		endpoint:   cfg.Endpoint,
	}, nil
}

// DownloadToTemp downloads a file from S3 to a temporary file on disk
// Returns the path to the temp file. Caller is responsible for deleting it.
func (c *Client) DownloadToTemp(ctx context.Context, bucket, key string) (string, error) {
	if bucket == "" {
		bucket = c.bucketName
	}

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

// UploadFromURL downloads an image from a URL and uploads it to S3
// Returns the S3 URL of the uploaded image
func (c *Client) UploadFromURL(ctx context.Context, imageURL, keyPrefix string) (string, error) {
	if imageURL == "" {
		return "", nil
	}

	// Download image from URL
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image body: %w", err)
	}

	// Determine content type and extension
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}

	ext := ".jpg"
	if strings.Contains(contentType, "png") {
		ext = ".png"
	} else if strings.Contains(contentType, "gif") {
		ext = ".gif"
	} else if strings.Contains(contentType, "webp") {
		ext = ".webp"
	}

	// Create S3 key: keyPrefix/timestamp.ext
	key := path.Join(keyPrefix, fmt.Sprintf("%d%s", time.Now().UnixNano(), ext))

	// Upload to S3
	_, err = c.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
		ACL:         "public-read", // Make images publicly accessible
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload image to S3: %w", err)
	}

	// Return public URL (virtual-hosted style for DO Spaces)
	// Format: https://{bucket}.{region}.digitaloceanspaces.com/{key}
	endpoint := strings.TrimPrefix(c.endpoint, "https://")
	s3URL := fmt.Sprintf("https://%s.%s/%s", c.bucketName, endpoint, key)
	return s3URL, nil
}

// GetS3Client returns the underlying S3 client
func (c *Client) GetS3Client() *s3.Client {
	return c.s3Client
}

// GetBucketName returns the configured bucket name
func (c *Client) GetBucketName() string {
	return c.bucketName
}
