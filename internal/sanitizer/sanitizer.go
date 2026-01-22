package sanitizer

// ChunkType represents the classification of a data chunk.
type ChunkType int

const (
	ChunkEmpty     ChunkType = iota
	ChunkJSONRPC             // Valid JSON-RPC message
	ChunkPollution           // Random text, logs, stack traces
)

// Sanitizer validates and cleans output from the child process.
// It ensures only valid JSON-RPC reaches the AI client.
type Sanitizer interface {
	// Classify determines if a byte slice is valid JSON-RPC or garbage.
	Classify(chunk []byte) ChunkType

	// ValidateUTF8 ensures the byte slice is valid UTF-8, replacing invalid bytes with replacement char.
	ValidateUTF8(data []byte) []byte
}
