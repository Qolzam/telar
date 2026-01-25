package chunker

import (
	"strings"
)

// Chunker splits text into semantically meaningful chunks with overlap
type Chunker struct {
	chunkSize    int // Approximate chunk size in characters
	chunkOverlap int // Overlap size in characters
}

// Config holds chunker configuration
type Config struct {
	ChunkSize    int // Approximate chunk size in characters (default: 4000)
	ChunkOverlap int // Overlap size in characters (default: 800)
}

// NewChunker creates a new chunker instance with the given configuration
func NewChunker(config Config) *Chunker {
	chunkSize := config.ChunkSize
	if chunkSize == 0 {
		chunkSize = 4000 // Default: ~1000 tokens ≈ 4000 chars
	}
	
	chunkOverlap := config.ChunkOverlap
	if chunkOverlap == 0 {
		chunkOverlap = 800 // Default: ~200 tokens ≈ 800 chars
	}
	
	// Ensure overlap is less than chunk size
	if chunkOverlap >= chunkSize {
		chunkOverlap = chunkSize / 5 // Default to 20% overlap
	}
	
	return &Chunker{
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
	}
}

// Split recursively splits text into chunks using a hierarchical approach:
// 1. Try to split by paragraphs (\n\n)
// 2. If chunk > size, split by lines (\n)
// 3. If chunk > size, split by sentences (. )
// 4. If chunk > size, split by words ( )
func (c *Chunker) Split(text string) []string {
	if len(text) <= c.chunkSize {
		return []string{text}
	}
	
	chunks := c.splitRecursive(text, c.chunkSize)
	
	// Apply overlap between chunks
	return c.applyOverlap(chunks)
}

// splitRecursive performs hierarchical text splitting
func (c *Chunker) splitRecursive(text string, maxSize int) []string {
	if len(text) <= maxSize {
		return []string{text}
	}
	
	// Try splitting by paragraphs first
	if chunks := c.splitByDelimiter(text, "\n\n", maxSize); len(chunks) > 1 {
		return chunks
	}
	
	// Try splitting by lines
	if chunks := c.splitByDelimiter(text, "\n", maxSize); len(chunks) > 1 {
		return chunks
	}
	
	// Try splitting by sentences
	if chunks := c.splitByDelimiter(text, ". ", maxSize); len(chunks) > 1 {
		return chunks
	}
	
	// Last resort: split by words
	return c.splitByDelimiter(text, " ", maxSize)
}

// splitByDelimiter splits text by a delimiter, ensuring no chunk exceeds maxSize
func (c *Chunker) splitByDelimiter(text, delimiter string, maxSize int) []string {
	var chunks []string
	parts := strings.Split(text, delimiter)
	
	currentChunk := ""
	
	for _, part := range parts {
		// If adding this part would exceed maxSize, save current chunk and start new one
		potentialChunk := currentChunk
		if currentChunk != "" {
			potentialChunk += delimiter
		}
		potentialChunk += part
		
		if len(potentialChunk) > maxSize && currentChunk != "" {
			chunks = append(chunks, currentChunk)
			currentChunk = part
		} else {
			currentChunk = potentialChunk
		}
	}
	
	// Add remaining chunk
	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}
	
	// If we couldn't split effectively, recursively split large chunks
	var finalChunks []string
	for _, chunk := range chunks {
		if len(chunk) > maxSize {
			// Recursively split this chunk using a more granular delimiter
			subChunks := c.splitRecursive(chunk, maxSize)
			finalChunks = append(finalChunks, subChunks...)
		} else {
			finalChunks = append(finalChunks, chunk)
		}
	}
	
	return finalChunks
}

// applyOverlap adds overlap between consecutive chunks to preserve context
func (c *Chunker) applyOverlap(chunks []string) []string {
	if len(chunks) <= 1 || c.chunkOverlap == 0 {
		return chunks
	}
	
	result := make([]string, 0, len(chunks))
	
	for i, chunk := range chunks {
		if i == 0 {
			result = append(result, chunk)
			continue
		}
		
		// Get overlap from previous chunk
		prevChunk := chunks[i-1]
		overlapStart := len(prevChunk) - c.chunkOverlap
		if overlapStart < 0 {
			overlapStart = 0
		}
		
		overlapText := prevChunk[overlapStart:]
		
		// Prepend overlap to current chunk
		chunkWithOverlap := overlapText + chunk
		result = append(result, chunkWithOverlap)
	}
	
	return result
}
