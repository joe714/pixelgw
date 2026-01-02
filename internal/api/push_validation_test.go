package api

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper to create minimal VP8L WebP with given dimensions
func createTestWebP(width, height int) []byte {
	buf := new(bytes.Buffer)

	// RIFF header
	buf.WriteString("RIFF")
	fileSizePlaceholder := buf.Len()
	buf.Write([]byte{0, 0, 0, 0})
	buf.WriteString("WEBP")

	// VP8L chunk
	buf.WriteString("VP8L")
	chunkSizePlaceholder := buf.Len()
	buf.Write([]byte{0, 0, 0, 0})

	// VP8L signature
	buf.WriteByte(0x2f)

	// Pack dimensions: (width-1) | ((height-1) << 14)
	bits := uint32(width-1) | (uint32(height-1) << 14)
	var packedDims [4]byte
	binary.LittleEndian.PutUint32(packedDims[:], bits)
	buf.Write(packedDims[:])

	// Minimal payload
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00})

	data := buf.Bytes()

	// Fill in sizes
	binary.LittleEndian.PutUint32(data[fileSizePlaceholder:], uint32(len(data)-8))
	chunkDataSize := len(data) - chunkSizePlaceholder - 4
	binary.LittleEndian.PutUint32(data[chunkSizePlaceholder:], uint32(chunkDataSize))

	return data
}

// Helper to create VP8 (lossy) WebP with given dimensions
func createTestWebPLossy(width, height int) []byte {
	buf := new(bytes.Buffer)

	// RIFF header
	buf.WriteString("RIFF")
	fileSizePlaceholder := buf.Len()
	buf.Write([]byte{0, 0, 0, 0})
	buf.WriteString("WEBP")

	// VP8 chunk (lossy)
	buf.WriteString("VP8 ")
	chunkSizePlaceholder := buf.Len()
	buf.Write([]byte{0, 0, 0, 0})

	// VP8 frame: 3-byte tag + frame header
	// Keyframe signature: 0x9d 0x01 0x2a
	buf.Write([]byte{0x9d, 0x01, 0x2a})

	// Width (14 bits) and height (14 bits), little-endian
	var widthBytes [2]byte
	var heightBytes [2]byte
	binary.LittleEndian.PutUint16(widthBytes[:], uint16(width))
	binary.LittleEndian.PutUint16(heightBytes[:], uint16(height))
	buf.Write(widthBytes[:])
	buf.Write(heightBytes[:])

	// Minimal VP8 data
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00})

	data := buf.Bytes()

	// Fill in sizes
	binary.LittleEndian.PutUint32(data[fileSizePlaceholder:], uint32(len(data)-8))
	chunkDataSize := len(data) - chunkSizePlaceholder - 4
	binary.LittleEndian.PutUint32(data[chunkSizePlaceholder:], uint32(chunkDataSize))

	return data
}

// Helper to create VP8X (extended) WebP with given dimensions
func createTestWebPExtended(width, height int) []byte {
	buf := new(bytes.Buffer)

	// RIFF header
	buf.WriteString("RIFF")
	fileSizePlaceholder := buf.Len()
	buf.Write([]byte{0, 0, 0, 0})
	buf.WriteString("WEBP")

	// VP8X chunk (extended format)
	buf.WriteString("VP8X")
	buf.Write([]byte{10, 0, 0, 0}) // Chunk size = 10 bytes

	// VP8X flags (1 byte)
	buf.WriteByte(0x00)

	// Reserved (3 bytes)
	buf.Write([]byte{0x00, 0x00, 0x00})

	// Canvas width - 1 (3 bytes, little-endian)
	w := width - 1
	buf.WriteByte(byte(w & 0xFF))
	buf.WriteByte(byte((w >> 8) & 0xFF))
	buf.WriteByte(byte((w >> 16) & 0xFF))

	// Canvas height - 1 (3 bytes, little-endian)
	h := height - 1
	buf.WriteByte(byte(h & 0xFF))
	buf.WriteByte(byte((h >> 8) & 0xFF))
	buf.WriteByte(byte((h >> 16) & 0xFF))

	data := buf.Bytes()

	// Fill in RIFF size
	binary.LittleEndian.PutUint32(data[fileSizePlaceholder:], uint32(len(data)-8))

	return data
}

func TestValidateWebP(t *testing.T) {
	t.Run("valid 64x32 VP8L", func(t *testing.T) {
		data := createTestWebP(64, 32)
		err := validateWebP(data)
		assert.NoError(t, err)
	})

	t.Run("valid 64x32 VP8 lossy", func(t *testing.T) {
		data := createTestWebPLossy(64, 32)
		err := validateWebP(data)
		assert.NoError(t, err)
	})

	t.Run("valid 64x32 VP8X extended", func(t *testing.T) {
		data := createTestWebPExtended(64, 32)
		err := validateWebP(data)
		assert.NoError(t, err)
	})

	t.Run("wrong dimensions 32x32", func(t *testing.T) {
		data := createTestWebP(32, 32)
		err := validateWebP(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "64x32")
	})

	t.Run("wrong dimensions 128x64", func(t *testing.T) {
		data := createTestWebP(128, 64)
		err := validateWebP(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "64x32")
	})

	t.Run("too small data", func(t *testing.T) {
		data := []byte("RIFF")
		err := validateWebP(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too small")
	})

	t.Run("missing RIFF header", func(t *testing.T) {
		data := []byte("XXXX....WEBP....")
		err := validateWebP(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "RIFF")
	})

	t.Run("missing WEBP marker", func(t *testing.T) {
		data := []byte("RIFF....XXXX....")
		err := validateWebP(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "WEBP")
	})

	t.Run("image too large", func(t *testing.T) {
		// Create valid header but data exceeds 128KB
		data := createTestWebP(64, 32)
		// Append extra data to exceed limit
		extra := make([]byte, 130*1024)
		data = append(data, extra...)
		err := validateWebP(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "128KB")
	})

	t.Run("not webp data", func(t *testing.T) {
		data := []byte("this is not a webp file at all, just random text")
		err := validateWebP(data)
		assert.Error(t, err)
	})

	t.Run("empty data", func(t *testing.T) {
		err := validateWebP([]byte{})
		assert.Error(t, err)
	})
}

func TestGetWebPDimensions(t *testing.T) {
	t.Run("VP8L format", func(t *testing.T) {
		data := createTestWebP(64, 32)
		w, h, err := getWebPDimensions(data)
		assert.NoError(t, err)
		assert.Equal(t, 64, w)
		assert.Equal(t, 32, h)
	})

	t.Run("VP8 lossy format", func(t *testing.T) {
		data := createTestWebPLossy(64, 32)
		w, h, err := getWebPDimensions(data)
		assert.NoError(t, err)
		assert.Equal(t, 64, w)
		assert.Equal(t, 32, h)
	})

	t.Run("VP8X extended format", func(t *testing.T) {
		data := createTestWebPExtended(64, 32)
		w, h, err := getWebPDimensions(data)
		assert.NoError(t, err)
		assert.Equal(t, 64, w)
		assert.Equal(t, 32, h)
	})

	t.Run("various VP8L dimensions", func(t *testing.T) {
		testCases := []struct {
			width, height int
		}{
			{1, 1},
			{100, 50},
			{320, 240},
			{1920, 1080},
		}

		for _, tc := range testCases {
			data := createTestWebP(tc.width, tc.height)
			w, h, err := getWebPDimensions(data)
			assert.NoError(t, err)
			assert.Equal(t, tc.width, w, "width mismatch for %dx%d", tc.width, tc.height)
			assert.Equal(t, tc.height, h, "height mismatch for %dx%d", tc.width, tc.height)
		}
	})

	t.Run("unsupported chunk type", func(t *testing.T) {
		// Create data with unknown chunk type
		data := []byte("RIFF....WEBPXXXX....")
		_, _, err := getWebPDimensions(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})

	t.Run("data too short for VP8L", func(t *testing.T) {
		// VP8L header but not enough data
		data := []byte("RIFF....WEBPVP8L....")
		_, _, err := getWebPDimensions(data)
		assert.Error(t, err)
	})
}

func TestValidationError(t *testing.T) {
	err := &validationError{"test message"}
	assert.Equal(t, "test message", err.Error())
}
