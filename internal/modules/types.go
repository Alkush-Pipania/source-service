package modules

// SourceProcessingMessage is the message received from the queue
type SourceProcessingMessage struct {
	SourceID string `json:"source_id"`
	Type     string `json:"type"` // "link", "note", "pdf", "ppt", "doc"
	UserID   string `json:"user_id"`
}

// SourceJob is the enriched job with full details from DB
type SourceJob struct {
	SourceID    string
	Type        string
	UserID      string
	OriginalURL string
	S3Bucket    string
	S3Key       string
	Title       string
}

// ProcessedContent holds the result of processing a source
type ProcessedContent struct {
	Title    string
	Text     string
	Metadata map[string]interface{}
}
