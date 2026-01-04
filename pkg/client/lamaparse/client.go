package lamaparse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	baseURL               = "https://api.cloud.llamaindex.ai/api/parsing"
	defaultPollInterval   = 2 * time.Second
	defaultMaxPollRetries = 150 // 5 minutes with 2s interval
	defaultTimeout        = 300 * time.Second
)

// ClientConfig holds configuration options for the LlamaParse client
type ClientConfig struct {
	APIKey         string
	Timeout        time.Duration
	PollInterval   time.Duration
	MaxPollRetries int
}

// Client is a LlamaParse API client
type Client struct {
	apiKey         string
	client         *http.Client
	pollInterval   time.Duration
	maxPollRetries int
}

// NewClient creates a new LlamaParse client with default settings
func NewClient(apiKey string) *Client {
	return NewClientWithConfig(ClientConfig{
		APIKey:         apiKey,
		Timeout:        defaultTimeout,
		PollInterval:   defaultPollInterval,
		MaxPollRetries: defaultMaxPollRetries,
	})
}

// NewClientWithConfig creates a new LlamaParse client with custom configuration
func NewClientWithConfig(cfg ClientConfig) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = defaultPollInterval
	}
	if cfg.MaxPollRetries == 0 {
		cfg.MaxPollRetries = defaultMaxPollRetries
	}

	return &Client{
		apiKey:         cfg.APIKey,
		client:         &http.Client{Timeout: cfg.Timeout},
		pollInterval:   cfg.PollInterval,
		maxPollRetries: cfg.MaxPollRetries,
	}
}

type uploadResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type jobResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"` // PENDING, SUCCESS, FAILED, ERROR
	Error  string `json:"error,omitempty"`
}

type markdownResponse struct {
	Markdown string `json:"markdown"`
}

type apiError struct {
	Detail string `json:"detail"`
}

// ParseFile uploads a file from disk and returns the extracted markdown
func (c *Client) ParseFile(ctx context.Context, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return c.ParseReader(ctx, file, filepath.Base(filePath))
}

// ParseBytes parses document content from bytes and returns the extracted markdown
func (c *Client) ParseBytes(ctx context.Context, data []byte, filename string) (string, error) {
	return c.ParseReader(ctx, bytes.NewReader(data), filename)
}

// ParseReader parses document content from an io.Reader and returns the extracted markdown
func (c *Client) ParseReader(ctx context.Context, reader io.Reader, filename string) (string, error) {
	// 1. Upload file
	jobID, err := c.upload(ctx, reader, filename)
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}

	// 2. Poll for completion
	if err := c.waitForJob(ctx, jobID); err != nil {
		return "", fmt.Errorf("job failed: %w", err)
	}

	// 3. Get result
	result, err := c.getResult(ctx, jobID)
	if err != nil {
		return "", fmt.Errorf("failed to get result: %w", err)
	}

	return result, nil
}

// ParseURL parses a document directly from a URL
func (c *Client) ParseURL(ctx context.Context, documentURL string) (string, error) {
	// Upload via URL endpoint
	payload := map[string]string{"url": documentURL}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/url", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", c.parseError(resp)
	}

	var res uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	// Poll for completion
	if err := c.waitForJob(ctx, res.ID); err != nil {
		return "", fmt.Errorf("job failed: %w", err)
	}

	return c.getResult(ctx, res.ID)
}

func (c *Client) upload(ctx context.Context, reader io.Reader, filename string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(part, reader); err != nil {
		return "", err
	}

	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/upload", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", c.parseError(resp)
	}

	var res uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.ID, nil
}

func (c *Client) waitForJob(ctx context.Context, jobID string) error {
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	retries := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := c.getJobStatus(ctx, jobID)
			if err != nil {
				return err
			}

			switch status.Status {
			case "SUCCESS":
				return nil
			case "FAILED", "ERROR":
				if status.Error != "" {
					return fmt.Errorf("parsing failed: %s", status.Error)
				}
				return fmt.Errorf("parsing failed with status: %s", status.Status)
			}

			retries++
			if retries >= c.maxPollRetries {
				return fmt.Errorf("max poll retries (%d) exceeded, job still pending", c.maxPollRetries)
			}
		}
	}
}

func (c *Client) getJobStatus(ctx context.Context, jobID string) (*jobResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/job/%s", baseURL, jobID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var res jobResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) getResult(ctx context.Context, jobID string) (string, error) {
	url := fmt.Sprintf("%s/job/%s/result/markdown", baseURL, jobID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", c.parseError(resp)
	}

	var res markdownResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.Markdown, nil
}

func (c *Client) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("lamaparse request failed with status %d", resp.StatusCode)
	}

	var apiErr apiError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Detail != "" {
		return fmt.Errorf("lamaparse error (status %d): %s", resp.StatusCode, apiErr.Detail)
	}

	return fmt.Errorf("lamaparse request failed (status %d): %s", resp.StatusCode, string(body))
}
