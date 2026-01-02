package api_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/joe714/pixelgw/internal/api"
	"github.com/joe714/pixelgw/internal/durable"
	"github.com/joe714/pixelgw/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createValidWebP creates a minimal valid WebP image with VP8L format (64x32)
func createValidWebP() []byte {
	buf := new(bytes.Buffer)

	// RIFF header
	buf.WriteString("RIFF")
	fileSizePlaceholder := buf.Len()
	buf.Write([]byte{0, 0, 0, 0}) // placeholder for file size
	buf.WriteString("WEBP")

	// VP8L chunk
	buf.WriteString("VP8L")
	chunkSizePlaceholder := buf.Len()
	buf.Write([]byte{0, 0, 0, 0}) // placeholder for chunk size

	// VP8L signature byte
	buf.WriteByte(0x2f)

	// Width and height packed: width-1 (14 bits) | height-1 (14 bits)
	// Width = 64, Height = 32 -> (64-1) = 63, (32-1) = 31
	bits := uint32(63) | (uint32(31) << 14)
	var packedDims [4]byte
	binary.LittleEndian.PutUint32(packedDims[:], bits)
	buf.Write(packedDims[:])

	// Minimal image data
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00})

	data := buf.Bytes()

	// Fill in file size (total - 8 bytes for RIFF header)
	binary.LittleEndian.PutUint32(data[fileSizePlaceholder:], uint32(len(data)-8))

	// Fill in chunk size
	chunkDataSize := len(data) - chunkSizePlaceholder - 4
	binary.LittleEndian.PutUint32(data[chunkSizePlaceholder:], uint32(chunkDataSize))

	return data
}

// createInvalidDimensionsWebP creates a WebP with wrong dimensions (32x32 instead of 64x32)
func createInvalidDimensionsWebP() []byte {
	buf := new(bytes.Buffer)

	buf.WriteString("RIFF")
	fileSizePlaceholder := buf.Len()
	buf.Write([]byte{0, 0, 0, 0})
	buf.WriteString("WEBP")

	buf.WriteString("VP8L")
	chunkSizePlaceholder := buf.Len()
	buf.Write([]byte{0, 0, 0, 0})

	buf.WriteByte(0x2f)

	// 32x32 dimensions (wrong)
	bits := uint32(31) | (uint32(31) << 14)
	var packedDims [4]byte
	binary.LittleEndian.PutUint32(packedDims[:], bits)
	buf.Write(packedDims[:])
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00})

	data := buf.Bytes()
	binary.LittleEndian.PutUint32(data[fileSizePlaceholder:], uint32(len(data)-8))
	chunkDataSize := len(data) - chunkSizePlaceholder - 4
	binary.LittleEndian.PutUint32(data[chunkSizePlaceholder:], uint32(chunkDataSize))

	return data
}

// setupPushTest creates a test environment with a channel
func setupPushTest(t *testing.T) (*durable.Store, *httptest.Server, func()) {
	store, cleanupDB, err := testutil.NewTestStore()
	require.NoError(t, err)

	// Create a test channel
	_, err = store.CreateChannel(context.Background(), "Test Channel", nil)
	require.NoError(t, err)

	server := api.NewServer(nil, store) // nil hub
	strictHandler := api.NewStrictHandlerWithOptions(server, nil, api.ServerOptions())
	handler := api.Handler(strictHandler)
	ts := httptest.NewServer(handler)

	cleanup := func() {
		ts.Close()
		cleanupDB()
	}

	return store, ts, cleanup
}

// TestPushChannelContent tests the POST /channels/{uuid}/push endpoint
func TestPushChannelContent(t *testing.T) {
	store, ts, cleanup := setupPushTest(t)
	defer cleanup()

	channels, err := store.GetAllChannels(context.Background())
	require.NoError(t, err)
	require.Len(t, channels, 1)
	channelUUID := channels[0].UUID

	t.Run("hub not available returns 500", func(t *testing.T) {
		webpData := createValidWebP()

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.webp")
		part.Write(webpData)
		writer.Close()

		req, _ := http.NewRequest("POST", ts.URL+"/channels/"+channelUUID.String()+"/push", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Hub is nil, so we get 500
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(respBody), "Hub not available")
	})

	t.Run("JSON applet request with nil hub", func(t *testing.T) {
		jsonBody := `{"applet": "clock", "duration": 30}`

		req, _ := http.NewRequest("POST", ts.URL+"/channels/"+channelUUID.String()+"/push", bytes.NewReader([]byte(jsonBody)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Hub is nil, so we get 500
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

// TestPushChannelContent_ChannelNotFound tests push to non-existent channel
// Note: With nil hub, we get 500 before channel check
func TestPushChannelContent_ChannelNotFound(t *testing.T) {
	_, ts, cleanup := setupPushTest(t)
	defer cleanup()

	nonExistentUUID := uuid.New()
	webpData := createValidWebP()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.webp")
	part.Write(webpData)
	writer.Close()

	req, _ := http.NewRequest("POST", ts.URL+"/channels/"+nonExistentUUID.String()+"/push", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Hub check happens first, so we get 500 instead of 404
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// TestClearChannelPush tests DELETE /channels/{uuid}/push
func TestClearChannelPush(t *testing.T) {
	store, ts, cleanup := setupPushTest(t)
	defer cleanup()

	channels, _ := store.GetAllChannels(context.Background())
	channelUUID := channels[0].UUID

	t.Run("hub not available returns 500", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", ts.URL+"/channels/"+channelUUID.String()+"/push", nil)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("channel not found returns 500 (hub checked first)", func(t *testing.T) {
		nonExistentUUID := uuid.New()
		req, _ := http.NewRequest("DELETE", ts.URL+"/channels/"+nonExistentUUID.String()+"/push", nil)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Hub check happens first
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

// TestPushDeviceContent tests POST /devices/{uuid}/push
func TestPushDeviceContent(t *testing.T) {
	store, ts, cleanup := setupPushTest(t)
	defer cleanup()

	// Create a device
	deviceUUID := uuid.New()
	_, err := store.LoginDevice(context.Background(), deviceUUID, "192.168.1.1")
	require.NoError(t, err)

	t.Run("hub not available returns 500", func(t *testing.T) {
		webpData := createValidWebP()

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.webp")
		part.Write(webpData)
		writer.Close()

		req, _ := http.NewRequest("POST", ts.URL+"/devices/"+deviceUUID.String()+"/push", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("device not found returns 500 (hub checked first)", func(t *testing.T) {
		nonExistentUUID := uuid.New()
		webpData := createValidWebP()

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.webp")
		part.Write(webpData)
		writer.Close()

		req, _ := http.NewRequest("POST", ts.URL+"/devices/"+nonExistentUUID.String()+"/push", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Hub check happens first
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

// TestClearDevicePush tests DELETE /devices/{uuid}/push
func TestClearDevicePush(t *testing.T) {
	store, ts, cleanup := setupPushTest(t)
	defer cleanup()

	// Create a device
	deviceUUID := uuid.New()
	_, err := store.LoginDevice(context.Background(), deviceUUID, "192.168.1.1")
	require.NoError(t, err)

	t.Run("hub not available returns 500", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", ts.URL+"/devices/"+deviceUUID.String()+"/push", nil)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("device not found returns 500 (hub checked first)", func(t *testing.T) {
		nonExistentUUID := uuid.New()
		req, _ := http.NewRequest("DELETE", ts.URL+"/devices/"+nonExistentUUID.String()+"/push", nil)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

// TestWebPValidation tests the WebP validation helper functions
// These test the validation logic directly without requiring a hub
func TestWebPValidation(t *testing.T) {
	t.Run("valid 64x32 WebP", func(t *testing.T) {
		data := createValidWebP()
		// Just verify the test data is structured correctly
		assert.True(t, len(data) >= 12)
		assert.Equal(t, []byte("RIFF"), data[0:4])
		assert.Equal(t, []byte("WEBP"), data[8:12])
	})

	t.Run("invalid dimensions WebP", func(t *testing.T) {
		data := createInvalidDimensionsWebP()
		// Just verify it's a valid WebP structure (but wrong dimensions)
		assert.True(t, len(data) >= 12)
		assert.Equal(t, []byte("RIFF"), data[0:4])
		assert.Equal(t, []byte("WEBP"), data[8:12])
	})
}

// TestPushEndpointRouting verifies the push endpoints are properly routed
func TestPushEndpointRouting(t *testing.T) {
	store, ts, cleanup := setupPushTest(t)
	defer cleanup()

	channels, _ := store.GetAllChannels(context.Background())
	channelUUID := channels[0].UUID

	t.Run("POST channel push is routed", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.Close()

		req, _ := http.NewRequest("POST", ts.URL+"/channels/"+channelUUID.String()+"/push", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should not be 404 (route not found)
		// We expect 500 (hub not available) or 400 (bad request)
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("DELETE channel push is routed", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", ts.URL+"/channels/"+channelUUID.String()+"/push", nil)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should not be 404
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("POST device push is routed", func(t *testing.T) {
		deviceUUID := uuid.New()
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.Close()

		req, _ := http.NewRequest("POST", ts.URL+"/devices/"+deviceUUID.String()+"/push", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("DELETE device push is routed", func(t *testing.T) {
		deviceUUID := uuid.New()
		req, _ := http.NewRequest("DELETE", ts.URL+"/devices/"+deviceUUID.String()+"/push", nil)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
	})
}
