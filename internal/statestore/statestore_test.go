package statestore

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateStore_SetAndGetInitialize(t *testing.T) {
	store := New()

	params := json.RawMessage(`{"capabilities":{"tools":true}}`)
	store.SetInitialize(params)

	result := store.GetInitialize()
	assert.Equal(t, params, result)
}

func TestStateStore_GetInitializeBeforeSet(t *testing.T) {
	store := New()

	result := store.GetInitialize()
	assert.Nil(t, result)
}

func TestStateStore_AddAndGetSubscriptions(t *testing.T) {
	store := New()

	store.AddSubscription("file://test1")
	store.AddSubscription("file://test2")

	subs := store.GetSubscriptions()
	assert.Len(t, subs, 2)
	assert.Contains(t, subs, "file://test1")
	assert.Contains(t, subs, "file://test2")
}

func TestStateStore_RemoveSubscription(t *testing.T) {
	store := New()

	store.AddSubscription("file://test1")
	store.AddSubscription("file://test2")

	store.RemoveSubscription("file://test1")

	subs := store.GetSubscriptions()
	assert.Len(t, subs, 1)
	assert.Contains(t, subs, "file://test2")
	assert.NotContains(t, subs, "file://test1")
}

func TestStateStore_UpdateSubscriptionID(t *testing.T) {
	store := New()

	store.AddSubscription("file://test")
	store.UpdateSubscriptionID("file://test", "sub-123")

	// Internal implementation detail, but we can test via re-add
	// which should maintain the ID
	subs := store.GetSubscriptions()
	assert.Contains(t, subs, "file://test")
}

func TestStateStore_Clear(t *testing.T) {
	store := New()

	params := json.RawMessage(`{"test":true}`)
	store.SetInitialize(params)
	store.AddSubscription("file://test1")
	store.AddSubscription("file://test2")

	store.Clear()

	assert.Nil(t, store.GetInitialize())
	assert.Empty(t, store.GetSubscriptions())
}

func TestStateStore_ConcurrentAccess(t *testing.T) {
	store := New()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			store.SetInitialize(json.RawMessage(`{"test":true}`))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			store.AddSubscription("file://concurrent")
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = store.GetSubscriptions()
		}
	}()

	wg.Wait()

	// Should not panic and should have valid state
	subs := store.GetSubscriptions()
	require.NotNil(t, subs)
}

func TestStateStore_DuplicateSubscriptions(t *testing.T) {
	store := New()

	store.AddSubscription("file://test")
	store.AddSubscription("file://test") // Add same URI again

	subs := store.GetSubscriptions()
	// Should only have one entry
	count := 0
	for _, sub := range subs {
		if sub == "file://test" {
			count++
		}
	}
	assert.Equal(t, 1, count, "duplicate subscription should not be added")
}

func TestStateStore_RemoveNonExistentSubscription(t *testing.T) {
	store := New()

	store.AddSubscription("file://exists")

	// Remove something that doesn't exist - should not panic
	store.RemoveSubscription("file://not-exists")

	subs := store.GetSubscriptions()
	assert.Len(t, subs, 1)
	assert.Contains(t, subs, "file://exists")
}
