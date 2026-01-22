package sanitizer

import (
	"bytes"
	"unicode/utf8"

	"github.com/tidwall/gjson"
)

type stdSanitizer struct{}

// New creates a new standard sanitizer.
func New() Sanitizer {
	return &stdSanitizer{}
}

func (s *stdSanitizer) Classify(chunk []byte) ChunkType {
	if len(chunk) == 0 {
		return ChunkEmpty
	}

	trimmed := bytes.TrimSpace(chunk)
	if len(trimmed) == 0 {
		return ChunkEmpty
	}

	// 1. Must be valid JSON
	if !gjson.ValidBytes(trimmed) {
		return ChunkPollution
	}

	// 2. Must be JSON-RPC 2.0
	// Check for "jsonrpc" field.
	// We parse it using gjson.GetBytes which is efficient.
	// Note: We are permissive here. If it's valid JSON but missing "jsonrpc", 
	// it might still be a response without the version field (rare but possible in some implementations),
	// OR it's just a random JSON object dumped by the user.
	// Strict MCP requires `jsonrpc: "2.0"`.
	res := gjson.GetBytes(trimmed, "jsonrpc")
	if res.Exists() && res.String() == "2.0" {
		return ChunkJSONRPC
	}

	// If it's a valid JSON array (batch request/response), we treat it as RPC for now
	if trimmed[0] == '[' {
		return ChunkJSONRPC
	}
	
	// If it's valid JSON but NOT RPC, it's pollution (e.g. console.log({"foo": "bar"}))
	return ChunkPollution
}

func (s *stdSanitizer) ValidateUTF8(data []byte) []byte {
	if utf8.Valid(data) {
		return data
	}

	// Allocate a new slice to avoid modifying the original in place if mostly valid
	// or just walk and replace.
	// For simplicity and safety, standard replacement:
	return bytes.ToValidUTF8(data, []byte("\ufffd"))
}
