package statestore

import (
	"encoding/json"
	"sync"
)

type store struct {
	mu               sync.RWMutex
	initializeParams json.RawMessage
	subscriptions    map[string]string // uri -> subscriptionID
}

// New creates a new StateStore instance
func New() StateStore {
	return &store{
		subscriptions: make(map[string]string),
	}
}

func (s *store) SetInitialize(params json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initializeParams = params
}

func (s *store) GetInitialize() json.RawMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initializeParams
}

func (s *store) AddSubscription(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Only add if not already present
	if _, exists := s.subscriptions[uri]; !exists {
		s.subscriptions[uri] = "" // Empty ID initially
	}
}

func (s *store) RemoveSubscription(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subscriptions, uri)
}

func (s *store) GetSubscriptions() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	uris := make([]string, 0, len(s.subscriptions))
	for uri := range s.subscriptions {
		uris = append(uris, uri)
	}
	return uris
}

func (s *store) UpdateSubscriptionID(uri string, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.subscriptions[uri]; exists {
		s.subscriptions[uri] = id
	}
}

func (s *store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initializeParams = nil
	s.subscriptions = make(map[string]string)
}
