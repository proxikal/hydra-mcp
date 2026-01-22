package statestore

import "encoding/json"

// StateStore manages MCP session state for crash recovery
type StateStore interface {
	SetInitialize(params json.RawMessage)
	GetInitialize() json.RawMessage

	AddSubscription(uri string)
	RemoveSubscription(uri string)
	GetSubscriptions() []string
	UpdateSubscriptionID(uri string, id string)

	Clear()
}
