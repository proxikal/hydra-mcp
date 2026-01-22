package logger

import (
	"os"

	"github.com/rs/zerolog"
)

// Logger defines the structured logging interface for Hydra.
// All implementations MUST write exclusively to stderr.
type Logger interface {
	Debug(msg string, props ...map[string]interface{})
	Info(msg string, props ...map[string]interface{})
	Warn(msg string, props ...map[string]interface{})
	Error(msg string, err error, props ...map[string]interface{})
}

type hydraLogger struct {
	z zerolog.Logger
}

// New creates a new structured logger writing to stderr.
func New(level string) Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	z := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(lvl)
	return &hydraLogger{z: z}
}

func (l *hydraLogger) Debug(msg string, props ...map[string]interface{}) {
	l.log(l.z.Debug(), msg, props...)
}

func (l *hydraLogger) Info(msg string, props ...map[string]interface{}) {
	l.log(l.z.Info(), msg, props...)
}

func (l *hydraLogger) Warn(msg string, props ...map[string]interface{}) {
	l.log(l.z.Warn(), msg, props...)
}

func (l *hydraLogger) Error(msg string, err error, props ...map[string]interface{}) {
	event := l.z.Error().Err(err)
	l.log(event, msg, props...)
}

func (l *hydraLogger) log(event *zerolog.Event, msg string, props ...map[string]interface{}) {
	if len(props) > 0 {
		for _, p := range props {
			event.Fields(p)
		}
	}
	event.Msg(msg)
}
