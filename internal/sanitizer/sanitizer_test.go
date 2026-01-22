package sanitizer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizer_Classify(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		input    string
		expected ChunkType
	}{
		{
			name:     "Valid Request",
			input:    `{"jsonrpc": "2.0", "method": "ping", "id": 1}`,
			expected: ChunkJSONRPC,
		},
		{
			name:     "Valid Response",
			input:    `{"jsonrpc": "2.0", "result": "pong", "id": 1}`,
			expected: ChunkJSONRPC,
		},
		{
			name:     "Pollution (Plain Text)",
			input:    `[DEBUG] Database connected`,
			expected: ChunkPollution,
		},
		{
			name:     "Pollution (Random JSON)",
			input:    `{"foo": "bar"}`, // Missing jsonrpc field
			expected: ChunkPollution,
		},
		{
			name:     "Empty",
			input:    `   `,
			expected: ChunkEmpty,
		},
		{
			name:     "Batch Request",
			input:    `[{"jsonrpc": "2.0", "method": "ping"}]`,
			expected: ChunkJSONRPC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, s.Classify([]byte(tt.input)))
		})
	}
}

func TestSanitizer_ValidateUTF8(t *testing.T) {
	s := New()

	// "Hello, World!" with invalid bytes in middle
	input := []byte("Hello, \xff\xfeWorld!")
	// Expect replacement char  (U+FFFD) - Go replaces the RUN of invalid bytes with one replacement.
	expected := []byte("Hello, \ufffdWorld!")

	result := s.ValidateUTF8(input)
	if !bytes.Equal(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}
