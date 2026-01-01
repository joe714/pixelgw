package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/joe714/pixelgw/internal/durable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChannels(t *testing.T) {
	store, ts, cleanup := setupTest()
	defer cleanup()

	ctx := context.Background()

	t.Run("empty database", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/channels", "")
		defer resp.Body.Close()

		// May return 200 with empty array or error for no rows
		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var channels []map[string]interface{}
			err = json.Unmarshal(body, &channels)
			require.NoError(t, err)
			assert.Empty(t, channels)
		}
	})

	t.Run("with channels", func(t *testing.T) {
		// Create some channels
		_, err := store.CreateChannel(ctx, "alpha-channel", nil)
		require.NoError(t, err)
		comment := "Test comment"
		_, err = store.CreateChannel(ctx, "beta-channel", &comment)
		require.NoError(t, err)

		resp := doRequest(ts, "GET", "/channels", "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var channels []map[string]interface{}
		err = json.Unmarshal(body, &channels)
		require.NoError(t, err)

		assert.Len(t, channels, 2)

		// Should be ordered by name
		assert.Equal(t, "alpha-channel", channels[0]["name"])
		assert.Equal(t, "beta-channel", channels[1]["name"])
		assert.Equal(t, "Test comment", channels[1]["comment"])
	})
}

func TestCreateChannel(t *testing.T) {
	_, ts, cleanup := setupTest()
	defer cleanup()

	t.Run("create channel", func(t *testing.T) {
		body := `{"name": "new-channel"}`
		resp := doRequest(ts, "POST", "/channels", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var channel map[string]interface{}
		err = json.Unmarshal(respBody, &channel)
		require.NoError(t, err)

		assert.Equal(t, "new-channel", channel["name"])
		assert.NotEmpty(t, channel["uuid"])
	})

	t.Run("create channel with comment", func(t *testing.T) {
		body := `{"name": "commented-channel", "comment": "This is a test"}`
		resp := doRequest(ts, "POST", "/channels", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var channel map[string]interface{}
		err = json.Unmarshal(respBody, &channel)
		require.NoError(t, err)

		assert.Equal(t, "commented-channel", channel["name"])
		assert.Equal(t, "This is a test", channel["comment"])
	})

	t.Run("duplicate channel fails", func(t *testing.T) {
		body := `{"name": "duplicate-channel"}`
		resp := doRequest(ts, "POST", "/channels", body)
		resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// Try to create again
		resp = doRequest(ts, "POST", "/channels", body)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("invalid request", func(t *testing.T) {
		resp := doRequest(ts, "POST", "/channels", "not json")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestFindChannelByUUID(t *testing.T) {
	store, ts, cleanup := setupTest()
	defer cleanup()

	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		ch, err := store.CreateChannel(ctx, "findable-channel", nil)
		require.NoError(t, err)

		resp := doRequest(ts, "GET", "/channels/"+ch.UUID.String(), "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var channel map[string]interface{}
		err = json.Unmarshal(body, &channel)
		require.NoError(t, err)

		assert.Equal(t, "findable-channel", channel["name"])
		assert.Equal(t, ch.UUID.String(), channel["uuid"])
	})

	t.Run("not found", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/channels/00000000-0000-0000-0000-000000000000", "")
		defer resp.Body.Close()

		// Should return error (either 404 or 500 depending on implementation)
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/channels/not-a-uuid", "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestChannelWithApplets(t *testing.T) {
	store, ts, cleanup := setupTest()
	defer cleanup()

	ctx := context.Background()

	// Create channel with applets
	ch, err := store.CreateChannel(ctx, "channel-with-applets", nil)
	require.NoError(t, err)

	// Add applets directly to the store (bypassing API which needs hub)
	app1 := &durable.ChannelApplet{AppID: "test-app-1", Idx: -1}
	err = store.CreateChannelApplet(ctx, ch.UUID, app1)
	require.NoError(t, err)

	app2 := &durable.ChannelApplet{AppID: "test-app-2", Idx: -1}
	err = store.CreateChannelApplet(ctx, ch.UUID, app2)
	require.NoError(t, err)

	t.Run("channel includes applets", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/channels/"+ch.UUID.String(), "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var channel map[string]interface{}
		err = json.Unmarshal(body, &channel)
		require.NoError(t, err)

		applets := channel["applets"].([]interface{})
		assert.Len(t, applets, 2)

		app1Data := applets[0].(map[string]interface{})
		assert.Equal(t, "test-app-1", app1Data["app-id"])
		assert.Equal(t, float64(0), app1Data["idx"])

		app2Data := applets[1].(map[string]interface{})
		assert.Equal(t, "test-app-2", app2Data["app-id"])
		assert.Equal(t, float64(1), app2Data["idx"])
	})
}
