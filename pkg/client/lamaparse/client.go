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

const baseURL = "https://api.cloud.llamaindex.ai/api/parsing"

type Client struct {
	apiKey string
	client *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		client: &http.Client{Timeout: 300 * time.Second}, // Parsing can take time
	}
}

type uploadResponse struct {
	ID string `json:"id"`
}

type jobResponse struct {
	Status string `json:"status"` // PENDING, SUCCESS, FAILED
	// Add other fields if needed
}

type markdownResponse struct {
	Markdown string `json:"markdown"`
}

// ParseFile uploads a file and returns the extracted markdown text
func (c *Client) ParseFile(ctx context.Context, filePath string) (string, error) {
	// 1. Upload File
	jobID, err := c.uploadFile(ctx, filePath)
	if err != nil {
		return "", err
	}

	// 2. Poll for Completion
	if err := c.waitForJob(ctx, jobID); err != nil {
		return "", err
	}

	// 3. Get Result
	return c.getResult(ctx, jobID)
}

func (c *Client) uploadFile(ctx context.Context, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", err
	}
	io.Copy(part, file)
	writer.Close()

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

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("lamaparse upload failed: %d", resp.StatusCode)
	}

	var res uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.ID, nil
}

func (c *Client) waitForJob(ctx context.Context, jobID string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/job/%s", baseURL, jobID), nil)
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
			
			resp, err := c.client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var res jobResponse
			if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
				return err
			}

			if res.Status == "SUCCESS" {
				return nil
			}
			if res.Status == "FAILED" {
				return fmt.Errorf("parsing job failed")
			}
		}
	}
}

func (c *Client) getResult(ctx context.Context, jobID string) (string, error) {
	url := fmt.Sprintf("%s/job/%s/result/markdown", baseURL, jobID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res markdownResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.Markdown, nil
}