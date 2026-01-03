package firmware

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	// MaxFirmwareSize is the maximum allowed firmware file size (2MB)
	MaxFirmwareSize = 2 * 1024 * 1024

	// PlatformESP32 is the platform identifier for ESP32 devices
	PlatformESP32 = "esp32"
)

var (
	ErrFileTooLarge     = errors.New("firmware file exceeds maximum size")
	ErrInvalidFirmware  = errors.New("invalid firmware binary")
	ErrDuplicateFirmware = errors.New("firmware with same SHA256 already exists for this platform")
	ErrFirmwareNotFound = errors.New("firmware not found")
)

// Firmware represents a firmware record in the database
type Firmware struct {
	UUID           uuid.UUID
	Platform       string
	Filename       string
	Description    *string
	Version        string
	BuildTimestamp time.Time
	ElfSHA256      string
	IDFVersion     string
	FileSize       int64
	IsDefault      bool
	UploadedAt     time.Time
}

// Extractor is the interface for extracting metadata from firmware binaries
type Extractor interface {
	Extract(data []byte) (*ESP32Metadata, error)
}

// ValidateFirmware validates firmware binary and extracts metadata
// Returns extracted metadata or error
func ValidateFirmware(data []byte, platform string) (*ESP32Metadata, error) {
	if len(data) > MaxFirmwareSize {
		return nil, ErrFileTooLarge
	}

	switch platform {
	case PlatformESP32:
		meta, err := ExtractESP32Metadata(data)
		if err != nil {
			return nil, errors.Join(ErrInvalidFirmware, err)
		}
		return meta, nil
	default:
		return nil, errors.New("unsupported platform: " + platform)
	}
}
