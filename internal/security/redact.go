package security

import "regexp"

type redactor struct {
	replacement string
}

// NewRedactor creates a new secret redactor using the default replacement.
func NewRedactor() Redactor {
	return &redactor{replacement: "[REDACTED]"}
}

// NewRedactorWithReplacement creates a redactor that uses a custom replacement
// string. If the provided replacement is empty, it falls back to the default.
func NewRedactorWithReplacement(replacement string) Redactor {
	if replacement == "" {
		replacement = "[REDACTED]"
	}
	return &redactor{replacement: replacement}
}

func (r *redactor) Redact(content string, patterns []string) string {
	result := content

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			// Skip invalid patterns
			continue
		}

		result = re.ReplaceAllString(result, r.replacement)
	}

	return result
}
