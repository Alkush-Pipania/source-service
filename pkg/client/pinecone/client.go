package pinecone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Vector represents a single record in Pinecone
type Vector struct {
	ID       string                 `json:"id"`
	Values   []float32              `json:"values"`
	Metadata map[string]interface{} `json:"metadata"`
}

type UpsertRequest struct {
	Vectors   []Vector `json:"vectors"`
	Namespace string   `json:"namespace,omitempty"`
}

type Client struct {
	apiKey  string
	hostURL string // e.g. "https://index-name-project-id.svc.pinecone.io"
	client  *http.Client
}

func NewClient(apiKey, hostURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		hostURL: hostURL,
		client:  &http.Client{},
	}
}

func (c *Client) Upsert(ctx context.Context, vectors []Vector) error {
	payload := UpsertRequest{
		Vectors: vectors,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/vectors/upsert", c.hostURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pinecone upsert failed with status: %d", resp.StatusCode)
	}

	return nil
}
