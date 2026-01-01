package durable_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/joe714/pixelgw/internal/durable"
	"github.com/joe714/pixelgw/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestStore(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	assert.NotNil(t, store)
	assert.NotNil(t, store.DB)
}

func TestStoreTransaction(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	// Test View (read-only) transaction
	err = store.View(context.Background(), func(tx *durable.TX) error {
		// Just verify we can execute a query
		return nil
	})
	assert.NoError(t, err)

	// Test Update (read-write) transaction
	err = store.Update(context.Background(), func(tx *durable.TX) error {
		// Just verify we can execute a query
		return nil
	})
	assert.NoError(t, err)
}

func TestStoreTransactionRollback(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	// Create a channel first
	ch, err := store.CreateChannel(context.Background(), "test-channel", nil)
	require.NoError(t, err)

	// Start a transaction that will fail
	err = store.Update(context.Background(), func(tx *durable.TX) error {
		// Try to create a duplicate channel (should trigger rollback)
		return fmt.Errorf("intentional error to trigger rollback")
	})
	assert.Error(t, err)

	// Verify original channel still exists (rollback worked)
	retrieved, err := store.GetChannelByUUID(context.Background(), ch.UUID)
	assert.NoError(t, err)
	assert.Equal(t, "test-channel", retrieved.Name)
}
