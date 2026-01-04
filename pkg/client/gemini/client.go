package gemini

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

const EmbeddingModel = "gemini-embedding-001"

var outputDimensionality int32 = 768

type Client struct {
	client *genai.Client
}

func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	if apiKey != "" {
		os.Setenv("GOOGLE_API_KEY", apiKey)
	}

	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	return &Client{
		client: client,
	}, nil
}

func (c *Client) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	contents := []*genai.Content{
		genai.NewContentFromText(text, genai.RoleUser),
	}

	result, err := c.client.Models.EmbedContent(ctx,
		EmbeddingModel,
		contents,
		&genai.EmbedContentConfig{OutputDimensionality: &outputDimensionality},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to embed content: %w", err)
	}

	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return result.Embeddings[0].Values, nil
}
