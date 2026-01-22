package injectable

import (
	"fmt"
	"sync"
)

// LogBuffer stores recent stderr lines.
type LogBuffer interface {
	Add(line string)
	GetRecent(n int) []string
}

type circularLogBuffer struct {
	mu    sync.Mutex
	lines []string
	max   int
}

// NewLogBuffer creates a bounded log buffer.
func NewLogBuffer(max int) LogBuffer {
	if max <= 0 {
		max = 500
	}
	return &circularLogBuffer{
		lines: make([]string, 0, max),
		max:   max,
	}
}

func (b *circularLogBuffer) Add(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.lines = append(b.lines, line)
	if len(b.lines) > b.max {
		b.lines = b.lines[len(b.lines)-b.max:]
	}
}

func (b *circularLogBuffer) GetRecent(n int) []string {
	b.mu.Lock()
	defer b.mu.Unlock()

	if n > b.max {
		n = b.max
	}

	if n <= 0 {
		return []string{}
	}

	if n > len(b.lines) {
		n = len(b.lines)
	}

	start := len(b.lines) - n
	result := make([]string, n)
	copy(result, b.lines[start:])
	return result
}

func (t *toolset) handleLogs(
	params map[string]interface{},
) (interface{}, error) {
	if t.buffer == nil {
		return nil, fmt.Errorf("log buffer not configured")
	}

	lines := 50
	if val, ok := params["lines"].(float64); ok {
		lines = int(val)
	}

	if lines < 1 {
		lines = 1
	}

	if lines > t.opts.MaxLogLines {
		lines = t.opts.MaxLogLines
	}

	logs := t.buffer.GetRecent(lines)

	for i, logLine := range logs {
		if t.redactor != nil && len(t.opts.RedactPatterns) > 0 {
			logs[i] = t.redactor.Redact(
				logLine,
				t.opts.RedactPatterns,
			)
		}
	}

	return map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	}, nil
}
