package utils

import (
	"strings"
)

// Chunk represents a piece of text with metadata
type Chunk struct {
	Text  string
	Index int
}

// SplitText splits a long string into chunks with overlap
// chunkSize: strict character limit (approx tokens)
// overlap: how many characters to repeat
func SplitText(text string, chunkSize int, overlap int) []Chunk {
	if chunkSize <= 0 {
		chunkSize = 1000
	}
	if overlap < 0 {
		overlap = 0
	}

	var chunks []Chunk
	runes := []rune(text) // Use runes to handle Emoji/Unicode safely
	length := len(runes)

	if length <= chunkSize {
		return []Chunk{{Text: text, Index: 0}}
	}

	for i := 0; i < length; i += (chunkSize - overlap) {
		end := i + chunkSize
		if end > length {
			end = length
		}

		chunkText := string(runes[i:end])
		// Optional: clean up hanging words at the cut?
		// For now, raw split is fine for embeddings.

		chunks = append(chunks, Chunk{
			Text:  strings.TrimSpace(chunkText),
			Index: len(chunks),
		})

		// Prevent infinite loop if overlap >= chunkSize
		if (chunkSize - overlap) <= 0 {
			break
		}
	}

	return chunks
}
