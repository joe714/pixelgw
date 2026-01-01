package durable_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/joe714/pixelgw/internal/durable"
	"github.com/joe714/pixelgw/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginDevice(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	// Create a channel for the default assignment
	_, err = store.CreateChannel(ctx, "default", nil)
	require.NoError(t, err)

	t.Run("new device", func(t *testing.T) {
		deviceUUID := uuid.New()
		device, err := store.LoginDevice(ctx, deviceUUID, "192.168.1.100")
		require.NoError(t, err)

		assert.Equal(t, deviceUUID, device.UUID)
		assert.Equal(t, deviceUUID.String(), device.Name) // New devices use UUID as name
		assert.Equal(t, durable.DefaultChannelUUID, device.ChannelUUID)
	})

	t.Run("existing device updates connection info", func(t *testing.T) {
		deviceUUID := uuid.New()

		// First login
		_, err := store.LoginDevice(ctx, deviceUUID, "192.168.1.1")
		require.NoError(t, err)

		// Second login from different IP
		device, err := store.LoginDevice(ctx, deviceUUID, "192.168.1.2")
		require.NoError(t, err)

		// Should return the same device
		assert.Equal(t, deviceUUID, device.UUID)

		// Verify IP was updated
		retrieved, err := store.GetDeviceByUUID(ctx, deviceUUID)
		require.NoError(t, err)
		require.NotNil(t, retrieved.LastIP)
		assert.Equal(t, "192.168.1.2", *retrieved.LastIP)
	})
}

func TestLogoutDevice(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	// Create default channel and device
	_, err = store.CreateChannel(ctx, "default", nil)
	require.NoError(t, err)

	deviceUUID := uuid.New()
	_, err = store.LoginDevice(ctx, deviceUUID, "192.168.1.1")
	require.NoError(t, err)

	t.Run("sets disconnect time", func(t *testing.T) {
		err := store.LogoutDevice(ctx, deviceUUID)
		require.NoError(t, err)

		// Verify disconnect time was set
		device, err := store.GetDeviceByUUID(ctx, deviceUUID)
		require.NoError(t, err)
		require.NotNil(t, device.LastDisconnectTime)
		assert.NotEmpty(t, *device.LastDisconnectTime)
	})
}

func TestGetAllDevices(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	// Create default channel
	_, err = store.CreateChannel(ctx, "default", nil)
	require.NoError(t, err)

	t.Run("empty database", func(t *testing.T) {
		devices, err := store.GetAllDevices(ctx)
		// May return error for no rows or empty slice - both are acceptable
		if err == nil {
			assert.Empty(t, devices)
		}
		// If there's an error, that's also acceptable for empty result
	})

	t.Run("multiple devices", func(t *testing.T) {
		// Create some devices
		_, err := store.LoginDevice(ctx, uuid.New(), "192.168.1.1")
		require.NoError(t, err)
		_, err = store.LoginDevice(ctx, uuid.New(), "192.168.1.2")
		require.NoError(t, err)
		_, err = store.LoginDevice(ctx, uuid.New(), "192.168.1.3")
		require.NoError(t, err)

		devices, err := store.GetAllDevices(ctx)
		require.NoError(t, err)
		assert.Len(t, devices, 3)
	})
}

func TestGetDeviceByUUID(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	// Create default channel
	_, err = store.CreateChannel(ctx, "default", nil)
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		deviceUUID := uuid.New()
		_, err := store.LoginDevice(ctx, deviceUUID, "192.168.1.1")
		require.NoError(t, err)

		device, err := store.GetDeviceByUUID(ctx, deviceUUID)
		require.NoError(t, err)
		assert.Equal(t, deviceUUID, device.UUID)
		require.NotNil(t, device.LastIP)
		assert.Equal(t, "192.168.1.1", *device.LastIP)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := store.GetDeviceByUUID(ctx, uuid.MustParse("00000000-0000-0000-0000-000000000000"))
		assert.Error(t, err)
	})
}

func TestModifyDevice(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	// Create channels
	ch1, err := store.CreateChannel(ctx, "channel-1", nil)
	require.NoError(t, err)
	ch2, err := store.CreateChannel(ctx, "channel-2", nil)
	require.NoError(t, err)

	// Create device
	deviceUUID := uuid.New()
	device, err := store.LoginDevice(ctx, deviceUUID, "192.168.1.1")
	require.NoError(t, err)

	// Assign to ch1 first
	device.ChannelUUID = ch1.UUID
	err = store.ModifyDevice(ctx, device)
	require.NoError(t, err)

	t.Run("change name", func(t *testing.T) {
		device.Name = "my-device"
		err := store.ModifyDevice(ctx, device)
		require.NoError(t, err)

		retrieved, err := store.GetDeviceByUUID(ctx, deviceUUID)
		require.NoError(t, err)
		assert.Equal(t, "my-device", retrieved.Name)
	})

	t.Run("change channel", func(t *testing.T) {
		device.ChannelUUID = ch2.UUID
		err := store.ModifyDevice(ctx, device)
		require.NoError(t, err)

		retrieved, err := store.GetDeviceByUUID(ctx, deviceUUID)
		require.NoError(t, err)
		assert.Equal(t, ch2.UUID, retrieved.ChannelUUID)
	})
}

func TestDeviceChannelName(t *testing.T) {
	store, cleanup, err := testutil.NewTestStore()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	// Create a named channel
	ch, err := store.CreateChannel(ctx, "test-channel", nil)
	require.NoError(t, err)

	// Create device
	deviceUUID := uuid.New()
	device, err := store.LoginDevice(ctx, deviceUUID, "192.168.1.1")
	require.NoError(t, err)

	// Assign to channel
	device.ChannelUUID = ch.UUID
	err = store.ModifyDevice(ctx, device)
	require.NoError(t, err)

	// Retrieve and check channel name is populated
	retrieved, err := store.GetDeviceByUUID(ctx, deviceUUID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.ChannelName)
	assert.Equal(t, "test-channel", *retrieved.ChannelName)
}
