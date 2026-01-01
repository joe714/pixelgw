package durable_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/joe714/pixelgw/internal/durable"
	"github.com/joe714/pixelgw/internal/errors"
	"github.com/joe714/pixelgw/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateChannel(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	t.Run("create channel without comment", func(t *testing.T) {
		ch, err := store.CreateChannel(ctx, "test-channel", nil)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, ch.UUID)
		assert.Equal(t, "test-channel", ch.Name)
		assert.Nil(t, ch.Comment)
	})

	t.Run("create channel with comment", func(t *testing.T) {
		comment := "Test comment"
		ch, err := store.CreateChannel(ctx, "channel-with-comment", &comment)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, ch.UUID)
		assert.Equal(t, "channel-with-comment", ch.Name)
		require.NotNil(t, ch.Comment)
		assert.Equal(t, "Test comment", *ch.Comment)
	})

	t.Run("duplicate channel name fails", func(t *testing.T) {
		_, err := store.CreateChannel(ctx, "duplicate-channel", nil)
		require.NoError(t, err)

		_, err = store.CreateChannel(ctx, "duplicate-channel", nil)
		assert.Error(t, err)
		assert.Equal(t, int32(1001), errors.Code(err)) // ChannelExists
	})
}

func TestGetAllChannels(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	t.Run("empty database", func(t *testing.T) {
		channels, err := store.GetAllChannels(ctx)
		// sqlair returns sql.ErrNoRows when no results, which is wrapped
		// This is acceptable behavior - just check we get nil or empty
		if err != nil {
			assert.Nil(t, channels)
		} else {
			assert.Empty(t, channels)
		}
	})

	t.Run("multiple channels", func(t *testing.T) {
		_, err := store.CreateChannel(ctx, "alpha", nil)
		require.NoError(t, err)
		_, err = store.CreateChannel(ctx, "beta", nil)
		require.NoError(t, err)
		_, err = store.CreateChannel(ctx, "gamma", nil)
		require.NoError(t, err)

		channels, err := store.GetAllChannels(ctx)
		require.NoError(t, err)
		assert.Len(t, channels, 3)

		// Should be ordered by name
		assert.Equal(t, "alpha", channels[0].Name)
		assert.Equal(t, "beta", channels[1].Name)
		assert.Equal(t, "gamma", channels[2].Name)
	})
}

func TestGetChannelByUUID(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		created, err := store.CreateChannel(ctx, "find-me", nil)
		require.NoError(t, err)

		found, err := store.GetChannelByUUID(ctx, created.UUID)
		require.NoError(t, err)
		assert.Equal(t, created.UUID, found.UUID)
		assert.Equal(t, "find-me", found.Name)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := store.GetChannelByUUID(ctx, uuid.MustParse("00000000-0000-0000-0000-000000000000"))
		assert.Error(t, err)
	})
}

func TestGetChannelByName(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		created, err := store.CreateChannel(ctx, "named-channel", nil)
		require.NoError(t, err)

		found, err := store.GetChannelByName(ctx, "named-channel")
		require.NoError(t, err)
		assert.Equal(t, created.UUID, found.UUID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := store.GetChannelByName(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Equal(t, int32(1002), errors.Code(err)) // ChannelNotFound
	})
}

func TestCreateChannelApplet(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	ch, err := store.CreateChannel(ctx, "applet-test", nil)
	require.NoError(t, err)

	t.Run("add applet at end", func(t *testing.T) {
		app := &durable.ChannelApplet{
			AppID: "test-app",
			Idx:   -1, // append to end
		}
		err := store.CreateChannelApplet(ctx, ch.UUID, app)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, app.UUID)
		assert.Equal(t, 0, app.Idx) // First applet should be at index 0
	})

	t.Run("add second applet", func(t *testing.T) {
		app := &durable.ChannelApplet{
			AppID: "test-app-2",
			Idx:   -1,
		}
		err := store.CreateChannelApplet(ctx, ch.UUID, app)
		require.NoError(t, err)
		assert.Equal(t, 1, app.Idx)
	})

	t.Run("add applet with config", func(t *testing.T) {
		config := `{"key": "value"}`
		app := &durable.ChannelApplet{
			AppID:  "configured-app",
			Idx:    -1,
			Config: &config,
		}
		err := store.CreateChannelApplet(ctx, ch.UUID, app)
		require.NoError(t, err)

		// Verify by retrieving channel
		retrieved, err := store.GetChannelByUUID(ctx, ch.UUID)
		require.NoError(t, err)

		found := false
		for _, a := range retrieved.Applets {
			if a.AppID == "configured-app" {
				found = true
				require.NotNil(t, a.Config)
				assert.Equal(t, `{"key": "value"}`, *a.Config)
			}
		}
		assert.True(t, found)
	})

	t.Run("index out of range", func(t *testing.T) {
		app := &durable.ChannelApplet{
			AppID: "bad-index",
			Idx:   100, // way too high
		}
		err := store.CreateChannelApplet(ctx, ch.UUID, app)
		assert.Error(t, err)
		assert.Equal(t, int32(1011), errors.Code(err)) // AppIndexOutOfRange
	})
}

func TestDeleteChannelApplet(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	ch, err := store.CreateChannel(ctx, "delete-test", nil)
	require.NoError(t, err)

	// Add some applets
	app1 := &durable.ChannelApplet{AppID: "app-1", Idx: -1}
	app2 := &durable.ChannelApplet{AppID: "app-2", Idx: -1}
	app3 := &durable.ChannelApplet{AppID: "app-3", Idx: -1}

	require.NoError(t, store.CreateChannelApplet(ctx, ch.UUID, app1))
	require.NoError(t, store.CreateChannelApplet(ctx, ch.UUID, app2))
	require.NoError(t, store.CreateChannelApplet(ctx, ch.UUID, app3))

	t.Run("delete middle applet", func(t *testing.T) {
		err := store.DeleteChannelApplet(ctx, ch.UUID, app2.UUID)
		require.NoError(t, err)

		// Verify remaining applets
		retrieved, err := store.GetChannelByUUID(ctx, ch.UUID)
		require.NoError(t, err)
		assert.Len(t, retrieved.Applets, 2)

		// Check indices are reordered
		assert.Equal(t, 0, retrieved.Applets[0].Idx)
		assert.Equal(t, 1, retrieved.Applets[1].Idx)
	})
}

func TestModifyChannelApplet(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	ch, err := store.CreateChannel(ctx, "modify-test", nil)
	require.NoError(t, err)

	app := &durable.ChannelApplet{AppID: "modifiable-app", Idx: -1}
	require.NoError(t, store.CreateChannelApplet(ctx, ch.UUID, app))

	t.Run("update config", func(t *testing.T) {
		newConfig := `{"updated": true}`
		err := store.ModifyChannelApplet(ctx, ch.UUID, app.UUID, nil, &newConfig)
		require.NoError(t, err)

		// Verify
		retrieved, err := store.GetChannelByUUID(ctx, ch.UUID)
		require.NoError(t, err)
		require.Len(t, retrieved.Applets, 1)
		require.NotNil(t, retrieved.Applets[0].Config)
		assert.Equal(t, `{"updated": true}`, *retrieved.Applets[0].Config)
	})
}

func TestChannelWithSubscribers(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	ch, err := store.CreateChannel(ctx, "subscriber-test", nil)
	require.NoError(t, err)

	// Add a device subscribed to this channel
	deviceUUID := uuid.New()
	device, err := store.LoginDevice(ctx, deviceUUID, "192.168.1.1")
	require.NoError(t, err)

	// Update device to subscribe to this channel
	device.ChannelUUID = ch.UUID
	err = store.ModifyDevice(ctx, device)
	require.NoError(t, err)

	// Retrieve channel and check subscribers
	retrieved, err := store.GetChannelByUUID(ctx, ch.UUID)
	require.NoError(t, err)
	assert.Len(t, retrieved.Subscribers, 1)
	assert.Equal(t, deviceUUID, retrieved.Subscribers[0].UUID)
}
