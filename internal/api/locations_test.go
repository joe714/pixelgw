package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchLocations(t *testing.T) {
	_, ts, cleanup := setupTest()
	defer cleanup()

	t.Run("search for new york", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/locations?q=new+york", "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var locations []map[string]interface{}
		err = json.Unmarshal(body, &locations)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(locations), 1)

		// Check first result has expected fields
		loc := locations[0]
		assert.Equal(t, "new-york-ny", loc["place_id"])
		assert.Equal(t, "New York, NY, USA", loc["description"])
		assert.NotEmpty(t, loc["lat"])
		assert.NotEmpty(t, loc["lng"])
		assert.NotEmpty(t, loc["timezone"])
	})

	t.Run("search with limit", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/locations?q=a&limit=3", "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var locations []map[string]interface{}
		err = json.Unmarshal(body, &locations)
		require.NoError(t, err)

		assert.LessOrEqual(t, len(locations), 3)
	})

	t.Run("search no results", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/locations?q=xyznonexistent", "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var locations []map[string]interface{}
		err = json.Unmarshal(body, &locations)
		require.NoError(t, err)

		assert.Empty(t, locations)
	})
}

func TestGetLocationByPlaceID(t *testing.T) {
	_, ts, cleanup := setupTest()
	defer cleanup()

	t.Run("found", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/locations/new-york-ny", "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var location map[string]interface{}
		err = json.Unmarshal(body, &location)
		require.NoError(t, err)

		assert.Equal(t, "new-york-ny", location["place_id"])
		assert.Equal(t, "New York, NY, USA", location["description"])
		assert.Equal(t, "New York", location["locality"])
		assert.Equal(t, "40.7128", location["lat"])
		assert.Equal(t, "-74.0060", location["lng"])
		assert.Equal(t, "America/New_York", location["timezone"])
	})

	t.Run("not found", func(t *testing.T) {
		resp := doRequest(ts, "GET", "/locations/nonexistent-place", "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
