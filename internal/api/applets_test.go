package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetApplets(t *testing.T) {
	_, ts, cleanup := setupTest()
	defer cleanup()

	t.Run("empty catalog (nil hub)", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/applets", "")
		defer resp.Body.Close()

		// With nil hub, should return empty array
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var applets []map[string]interface{}
		err = json.Unmarshal(body, &applets)
		require.NoError(t, err)
		assert.Empty(t, applets)
	})
}

func TestGetAppletByID(t *testing.T) {
	_, ts, cleanup := setupTest()
	defer cleanup()

	t.Run("catalog not available", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/applets/some-applet-id", "")
		defer resp.Body.Close()

		// With nil hub, should return error
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestRenderApplet(t *testing.T) {
	_, ts, cleanup := setupTest()
	defer cleanup()

	t.Run("catalog not available", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/applets/some-applet-id/render", "")
		defer resp.Body.Close()

		// With nil hub, should return 404
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var errResp map[string]interface{}
		err = json.Unmarshal(body, &errResp)
		require.NoError(t, err)
		assert.Equal(t, "applet catalog not available", errResp["message"])
	})
}
