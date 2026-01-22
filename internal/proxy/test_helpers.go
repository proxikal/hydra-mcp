package proxy

import (
	"io"
	"time"

	"github.com/proxikal/hydra/internal/recorder"
	"github.com/proxikal/hydra/internal/transport"
	"github.com/stretchr/testify/mock"
)

type stubTransport struct {
	writes [][]byte
	readCh chan []byte
	err    error
}

func (s *stubTransport) Read() ([]byte, error) {
	if s.err != nil {
		err := s.err
		s.err = nil
		return nil, err
	}
	if s.readCh == nil {
		return nil, io.EOF
	}
	msg, ok := <-s.readCh
	if !ok {
		return nil, io.EOF
	}
	return msg, nil
}

func (s *stubTransport) Write(p []byte) error {
	s.writes = append(s.writes, append([]byte{}, p...))
	return nil
}

func (s *stubTransport) Close() error { return nil }

func (s *stubTransport) DetectProtocol(time.Duration) (transport.Protocol, error) {
	return transport.ProtocolNDJSON, nil
}

type stubLogger struct{}

func (stubLogger) Debug(string, ...map[string]interface{}) {}
func (stubLogger) Info(string, ...map[string]interface{})  {}
func (stubLogger) Warn(string, ...map[string]interface{})  {}
func (stubLogger) Error(string, error, ...map[string]interface{}) {
}

type mockRecorder struct {
	mock.Mock
}

func (m *mockRecorder) RecordRequest(direction string, msg recorder.JSONRPCMessage) {
	m.Called(direction, msg)
}

func (m *mockRecorder) RecordResponse(direction string, msg recorder.JSONRPCMessage) {
	m.Called(direction, msg)
}

func (m *mockRecorder) Export(path string) error {
	args := m.Called(path)
	return args.Error(0)
}
