package logger

import (
	"bytes"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	// Create a raw zerolog to a buffer for testing, 
	// though the real New() always goes to stderr.
	z := zerolog.New(&buf).Level(zerolog.DebugLevel)
	l := &hydraLogger{z: z}

	t.Run("Info Level", func(t *testing.T) {
		buf.Reset()
		l.Info("test message", map[string]interface{}{"key": "value"})
		assert.Contains(t, buf.String(), `"level":"info"`)
		assert.Contains(t, buf.String(), `"message":"test message"`)
		assert.Contains(t, buf.String(), `"key":"value"`)
	})

	t.Run("Error Level with Err", func(t *testing.T) {
		buf.Reset()
		err := errors.New("boom")
		l.Error("failed op", err, map[string]interface{}{"id": 123})
		assert.Contains(t, buf.String(), `"level":"error"`)
		assert.Contains(t, buf.String(), `"error":"boom"`)
		assert.Contains(t, buf.String(), `"message":"failed op"`)
		assert.Contains(t, buf.String(), `"id":123`)
	})

	t.Run("Debug Level Filtering", func(t *testing.T) {
		buf.Reset()
		// New logger at Info level
		infoLogger := New("info").(*hydraLogger)
		infoLogger.z = infoLogger.z.Output(&buf)
		
		infoLogger.Debug("should not see this")
		assert.Empty(t, buf.String())

		infoLogger.Info("should see this")
		assert.NotEmpty(t, buf.String())
	})
}
