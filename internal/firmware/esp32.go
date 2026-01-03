package firmware

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

const (
	// ESP32 app descriptor magic word
	esp32Magic = 0xABCD5432

	// Offsets within the ESP32 binary (after the image header)
	magicOffset      = 0x20
	versionOffset    = 0x30
	buildTimeOffset  = 0x70
	buildDateOffset  = 0x80
	idfVersionOffset = 0x90
	elfSHA256Offset  = 0xB0

	// Field sizes
	versionSize    = 32
	buildTimeSize  = 16
	buildDateSize  = 16
	idfVersionSize = 32
	elfSHA256Size  = 32

	// Minimum file size to contain metadata
	minFileSize = 0xD0 // elfSHA256Offset + elfSHA256Size
)

// ESP32Metadata contains extracted metadata from an ESP32 firmware binary
type ESP32Metadata struct {
	Version        string
	BuildTimestamp time.Time
	IDFVersion     string
	ElfSHA256      string // hex encoded, 64 chars
}

// ExtractESP32Metadata extracts metadata from an ESP32 firmware binary
func ExtractESP32Metadata(data []byte) (*ESP32Metadata, error) {
	if len(data) < minFileSize {
		return nil, errors.New("file too small to be valid ESP32 firmware")
	}

	// Validate magic word
	magic := binary.LittleEndian.Uint32(data[magicOffset : magicOffset+4])
	if magic != esp32Magic {
		return nil, fmt.Errorf("invalid ESP32 binary: wrong magic (got 0x%08X, expected 0x%08X)", magic, esp32Magic)
	}

	// Extract version (null-terminated string)
	version := extractNullTerminatedString(data[versionOffset : versionOffset+versionSize])

	// Extract build time and date
	buildTime := extractNullTerminatedString(data[buildTimeOffset : buildTimeOffset+buildTimeSize])
	buildDate := extractNullTerminatedString(data[buildDateOffset : buildDateOffset+buildDateSize])

	// Parse and combine build date + time into ISO 8601 timestamp
	buildTimestamp, err := parseBuildTimestamp(buildDate, buildTime)
	if err != nil {
		// If parsing fails, use zero time but don't fail extraction
		buildTimestamp = time.Time{}
	}

	// Extract IDF version
	idfVersion := extractNullTerminatedString(data[idfVersionOffset : idfVersionOffset+idfVersionSize])

	// Extract ELF SHA256 (binary -> hex)
	elfSHA256 := hex.EncodeToString(data[elfSHA256Offset : elfSHA256Offset+elfSHA256Size])

	return &ESP32Metadata{
		Version:        version,
		BuildTimestamp: buildTimestamp,
		IDFVersion:     idfVersion,
		ElfSHA256:      elfSHA256,
	}, nil
}

// extractNullTerminatedString extracts a null-terminated string from a byte slice
func extractNullTerminatedString(data []byte) string {
	for i, b := range data {
		if b == 0 {
			return string(data[:i])
		}
	}
	return string(data)
}

// parseBuildTimestamp parses ESP-IDF build date and time into a time.Time
// Date format: "Jan  3 2026" or "Jan 15 2026"
// Time format: "04:52:44"
func parseBuildTimestamp(dateStr, timeStr string) (time.Time, error) {
	combined := dateStr + " " + timeStr
	// Try parsing with single-digit day (space padded)
	t, err := time.Parse("Jan  2 2006 15:04:05", combined)
	if err != nil {
		// Try parsing with double-digit day
		t, err = time.Parse("Jan 2 2006 15:04:05", combined)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse build timestamp: %w", err)
	}
	return t.UTC(), nil
}
