package pinecone

import (
	"context"
	"fmt"

	"github.com/pinecone-io/go-pinecone/v4/pinecone"
	"google.golang.org/protobuf/types/known/structpb"
)

// Vector represents a single record in Pinecone
type Vector struct {
	ID       string
	Values   []float32
	Metadata map[string]interface{}
}

// Client wraps the Pinecone SDK client
type Client struct {
	pc      *pinecone.Client
	idxConn *pinecone.IndexConnection
}

// NewClient creates a new Pinecone client connected to the specified index
func NewClient(apiKey, hostURL string) (*Client, error) {
	pc, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pinecone client: %w", err)
	}

	idxConn, err := pc.Index(pinecone.NewIndexConnParams{
		Host: hostURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to index: %w", err)
	}

	return &Client{
		pc:      pc,
		idxConn: idxConn,
	}, nil
}

// Upsert inserts or updates vectors in the index
func (c *Client) Upsert(ctx context.Context, vectors []Vector) (uint32, error) {
	if len(vectors) == 0 {
		return 0, nil
	}

	pcVectors := make([]*pinecone.Vector, len(vectors))
	for i, v := range vectors {
		var metadata *structpb.Struct
		if v.Metadata != nil {
			var err error
			metadata, err = structpb.NewStruct(v.Metadata)
			if err != nil {
				return 0, fmt.Errorf("failed to create metadata for vector %s: %w", v.ID, err)
			}
		}

		values := v.Values // local copy for pointer
		pcVectors[i] = &pinecone.Vector{
			Id:       v.ID,
			Values:   &values,
			Metadata: metadata,
		}
	}

	count, err := c.idxConn.UpsertVectors(ctx, pcVectors)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert vectors: %w", err)
	}

	return count, nil
}

// UpsertWithNamespace inserts or updates vectors in a specific namespace (e.g., userID)
func (c *Client) UpsertWithNamespace(ctx context.Context, namespace string, vectors []Vector) (uint32, error) {
	if len(vectors) == 0 {
		return 0, nil
	}

	// Create a namespaced connection
	namespacedConn := c.idxConn.WithNamespace(namespace)

	pcVectors := make([]*pinecone.Vector, len(vectors))
	for i, v := range vectors {
		var metadata *structpb.Struct
		if v.Metadata != nil {
			var err error
			metadata, err = structpb.NewStruct(v.Metadata)
			if err != nil {
				return 0, fmt.Errorf("failed to create metadata for vector %s: %w", v.ID, err)
			}
		}

		values := v.Values // local copy for pointer
		pcVectors[i] = &pinecone.Vector{
			Id:       v.ID,
			Values:   &values,
			Metadata: metadata,
		}
	}

	count, err := namespacedConn.UpsertVectors(ctx, pcVectors)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert vectors to namespace %s: %w", namespace, err)
	}

	return count, nil
}

// UpsertBatch upserts vectors in batches of specified size (max 1000 recommended)
func (c *Client) UpsertBatch(ctx context.Context, vectors []Vector, batchSize int) (uint32, error) {
	if batchSize <= 0 {
		batchSize = 100 // default batch size
	}
	if batchSize > 1000 {
		batchSize = 1000 // max recommended by Pinecone
	}

	var totalCount uint32
	for i := 0; i < len(vectors); i += batchSize {
		end := i + batchSize
		if end > len(vectors) {
			end = len(vectors)
		}

		count, err := c.Upsert(ctx, vectors[i:end])
		if err != nil {
			return totalCount, fmt.Errorf("batch upsert failed at index %d: %w", i, err)
		}
		totalCount += count
	}

	return totalCount, nil
}

// Close closes the index connection
func (c *Client) Close() error {
	if c.idxConn != nil {
		return c.idxConn.Close()
	}
	return nil
}
