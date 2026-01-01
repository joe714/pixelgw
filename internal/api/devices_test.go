package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDevices(t *testing.T) {
	store, ts, cleanup := setupTest()
	defer cleanup()

	ctx := context.Background()

	t.Run("empty database", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/devices", "")
		defer resp.Body.Close()

		// May return 200 with empty array or error for no rows
		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var devices []map[string]interface{}
			err = json.Unmarshal(body, &devices)
			require.NoError(t, err)
			assert.Empty(t, devices)
		}
	})

	t.Run("with devices", func(t *testing.T) {
		// Create a channel first (devices need a channel)
		ch, err := store.CreateChannel(ctx, "test-channel", nil)
		require.NoError(t, err)

		// Login a device (this creates the device)
		_, err = store.LoginDevice(ctx, testDeviceUUID1, "192.168.1.100")
		require.NoError(t, err)

		// Assign device to channel
		d, err := store.GetDeviceByUUID(ctx, testDeviceUUID1)
		require.NoError(t, err)
		d.ChannelUUID = ch.UUID
		err = store.ModifyDevice(ctx, d)
		require.NoError(t, err)

		resp := doRequest(ts, "GET", "/devices", "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var devices []map[string]interface{}
		err = json.Unmarshal(body, &devices)
		require.NoError(t, err)

		assert.Len(t, devices, 1)
		assert.Equal(t, testDeviceUUID1.String(), devices[0]["uuid"])
		assert.Equal(t, false, devices[0]["connected"])
		assert.Equal(t, "192.168.1.100", devices[0]["last-ip"])
	})
}

func TestGetDeviceByUUID(t *testing.T) {
	store, ts, cleanup := setupTest()
	defer cleanup()

	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		// Create a channel first
		ch, err := store.CreateChannel(ctx, "device-channel", nil)
		require.NoError(t, err)

		// Login a device
		_, err = store.LoginDevice(ctx, testDeviceUUID2, "10.0.0.1")
		require.NoError(t, err)

		// Assign device to channel
		d, err := store.GetDeviceByUUID(ctx, testDeviceUUID2)
		require.NoError(t, err)
		d.ChannelUUID = ch.UUID
		d.Name = "My Test Device"
		err = store.ModifyDevice(ctx, d)
		require.NoError(t, err)

		resp := doRequest(ts, "GET", "/devices/"+testDeviceUUID2.String(), "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var device map[string]interface{}
		err = json.Unmarshal(body, &device)
		require.NoError(t, err)

		assert.Equal(t, testDeviceUUID2.String(), device["uuid"])
		assert.Equal(t, "My Test Device", device["name"])
		assert.Equal(t, false, device["connected"])

		channel := device["channel"].(map[string]interface{})
		assert.Equal(t, ch.UUID.String(), channel["uuid"])
		assert.Equal(t, "device-channel", channel["name"])
	})

	t.Run("not found", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/devices/00000000-0000-0000-0000-000000000000", "")
		defer resp.Body.Close()

		// Should return error (either 404 or 500 depending on implementation)
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/devices/not-a-uuid", "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestPatchDevice(t *testing.T) {
	store, ts, cleanup := setupTest()
	defer cleanup()

	ctx := context.Background()

	// Setup: create channels and device
	ch1, err := store.CreateChannel(ctx, "channel-one", nil)
	require.NoError(t, err)
	ch2, err := store.CreateChannel(ctx, "channel-two", nil)
	require.NoError(t, err)

	_, err = store.LoginDevice(ctx, testDeviceUUID3, "172.16.0.1")
	require.NoError(t, err)
	d, err := store.GetDeviceByUUID(ctx, testDeviceUUID3)
	require.NoError(t, err)
	d.ChannelUUID = ch1.UUID
	err = store.ModifyDevice(ctx, d)
	require.NoError(t, err)

	t.Run("rename device", func(t *testing.T) {
		body := `{"name": "Updated Device Name"}`
		resp := doRequest(ts, "PATCH", "/devices/"+testDeviceUUID3.String(), body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify the change
		d, err := store.GetDeviceByUUID(ctx, testDeviceUUID3)
		require.NoError(t, err)
		assert.Equal(t, "Updated Device Name", d.Name)
	})

	t.Run("change channel by uuid", func(t *testing.T) {
		body := `{"channel": {"uuid": "` + ch2.UUID.String() + `"}}`
		resp := doRequest(ts, "PATCH", "/devices/"+testDeviceUUID3.String(), body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify the change
		d, err := store.GetDeviceByUUID(ctx, testDeviceUUID3)
		require.NoError(t, err)
		assert.Equal(t, ch2.UUID, d.ChannelUUID)
	})

	t.Run("change channel by name", func(t *testing.T) {
		body := `{"channel": {"name": "channel-one"}}`
		resp := doRequest(ts, "PATCH", "/devices/"+testDeviceUUID3.String(), body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify the change
		d, err := store.GetDeviceByUUID(ctx, testDeviceUUID3)
		require.NoError(t, err)
		assert.Equal(t, ch1.UUID, d.ChannelUUID)
	})

	t.Run("no attributes provided", func(t *testing.T) {
		resp := doRequest(ts, "PATCH", "/devices/"+testDeviceUUID3.String(), "{}")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("device not found", func(t *testing.T) {
		body := `{"name": "New Name"}`
		resp := doRequest(ts, "PATCH", "/devices/00000000-0000-0000-0000-000000000000", body)
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("channel not found", func(t *testing.T) {
		body := `{"channel": {"uuid": "99999999-9999-9999-9999-999999999999"}}`
		resp := doRequest(ts, "PATCH", "/devices/"+testDeviceUUID3.String(), body)
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
	})
}
