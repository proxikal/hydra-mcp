package security

import "regexp"

type redactor struct{}

// NewRedactor creates a new secret redactor
func NewRedactor() Redactor {
	return &redactor{}
}

func (r *redactor) Redact(content string, patterns []string) string {
	result := content

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			// Skip invalid patterns
			continue
		}

		result = re.ReplaceAllString(result, "[REDACTED]")
	}

	return result
}
