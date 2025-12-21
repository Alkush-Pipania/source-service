package gemini

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Client struct {
	emModel *genai.EmbeddingModel
	client  *genai.Client
}

func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	// 'text-embedding-004' is the latest, or use 'embedding-001'
	emModel := client.EmbeddingModel("text-embedding-004")
	return &Client{
		client:  client,
		emModel: emModel,
	}, nil
}

func (c *Client) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	res, err := c.emModel.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, err
	}

	if res.Embedding == nil {
		return nil, fmt.Errorf("failed to generate embedding")
	}

	return res.Embedding.Values, nil
}

func (c *Client) Close() {
	c.client.Close()
}
