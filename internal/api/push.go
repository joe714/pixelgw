package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	ne "github.com/joe714/pixelgw/internal/errors"
)

const (
	maxImageSize   = 128 * 1024 // 128KB
	defaultDuration = 15        // seconds
)

// WebP validation
var webpMagic = []byte("RIFF")
var webpFormat = []byte("WEBP")

// validateWebP checks if the data is a valid WebP image with correct dimensions
func validateWebP(data []byte) error {
	if len(data) < 12 {
		return &validationError{"image too small to be valid WebP"}
	}

	// Check RIFF header
	if !bytes.Equal(data[0:4], webpMagic) {
		return &validationError{"invalid WebP: missing RIFF header"}
	}

	// Check WEBP format
	if !bytes.Equal(data[8:12], webpFormat) {
		return &validationError{"invalid WebP: missing WEBP format marker"}
	}

	// Check size
	if len(data) > maxImageSize {
		return &validationError{"image exceeds maximum size of 128KB"}
	}

	// Parse dimensions from VP8/VP8L/VP8X chunk
	width, height, err := getWebPDimensions(data)
	if err != nil {
		return err
	}

	// Validate dimensions (must be 64x32)
	if width != 64 || height != 32 {
		return &validationError{"image must be 64x32 pixels"}
	}

	return nil
}

// getWebPDimensions extracts width and height from a WebP file
func getWebPDimensions(data []byte) (int, int, error) {
	if len(data) < 16 {
		return 0, 0, &validationError{"WebP data too short"}
	}

	// Skip RIFF header (12 bytes) and look at chunk type
	chunkType := string(data[12:16])

	switch chunkType {
	case "VP8 ":
		// Lossy WebP
		if len(data) < 30 {
			return 0, 0, &validationError{"VP8 data too short"}
		}
		// VP8 bitstream starts at offset 20, frame tag at 23
		// Width and height are at specific offsets in the frame header
		// Skip chunk header (8 bytes) and frame tag (3 bytes)
		frameStart := 20
		if len(data) < frameStart+10 {
			return 0, 0, &validationError{"VP8 frame data too short"}
		}
		// Check for VP8 key frame signature (0x9d 0x01 0x2a)
		if data[frameStart] != 0x9d || data[frameStart+1] != 0x01 || data[frameStart+2] != 0x2a {
			return 0, 0, &validationError{"invalid VP8 frame signature"}
		}
		// Width and height are 14 bits each, little-endian
		width := int(binary.LittleEndian.Uint16(data[frameStart+3:frameStart+5])) & 0x3FFF
		height := int(binary.LittleEndian.Uint16(data[frameStart+5:frameStart+7])) & 0x3FFF
		return width, height, nil

	case "VP8L":
		// Lossless WebP
		if len(data) < 25 {
			return 0, 0, &validationError{"VP8L data too short"}
		}
		// VP8L signature byte at offset 20
		if data[20] != 0x2f {
			return 0, 0, &validationError{"invalid VP8L signature"}
		}
		// Width and height are packed in next 4 bytes
		bits := binary.LittleEndian.Uint32(data[21:25])
		width := int(bits&0x3FFF) + 1
		height := int((bits>>14)&0x3FFF) + 1
		return width, height, nil

	case "VP8X":
		// Extended WebP
		if len(data) < 30 {
			return 0, 0, &validationError{"VP8X data too short"}
		}
		// Width and height are 24 bits each at specific offsets
		// Canvas width at offset 24 (3 bytes, little-endian, +1)
		// Canvas height at offset 27 (3 bytes, little-endian, +1)
		width := int(data[24]) | int(data[25])<<8 | int(data[26])<<16
		height := int(data[27]) | int(data[28])<<8 | int(data[29])<<16
		return width + 1, height + 1, nil

	default:
		return 0, 0, &validationError{"unsupported WebP format: " + chunkType}
	}
}

type validationError struct {
	message string
}

func (e *validationError) Error() string {
	return e.message
}

// PushChannelContent handles POST /channels/{uuid}/push
func (s *Server) PushChannelContent(ctx context.Context, request PushChannelContentRequestObject) (PushChannelContentResponseObject, error) {
	// Check if hub is available
	if s.hub == nil {
		return PushChannelContentdefaultJSONResponse{
			Body:       Error{Code: http.StatusInternalServerError, Message: "Hub not available"},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	// Verify channel exists
	_, err := s.store.GetChannelByUUID(ctx, request.UUID)
	if err != nil {
		if ne.Is(err, ne.ChannelNotFound) {
			return PushChannelContent404JSONResponse{
				Code:    http.StatusNotFound,
				Message: "Channel not found",
			}, nil
		}
		return PushChannelContentdefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	var imageData []byte
	var duration time.Duration

	if request.JSONBody != nil {
		// Applet render request
		imageData, duration, err = s.renderAppletForPush(request.JSONBody)
		if err != nil {
			return PushChannelContent400JSONResponse{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			}, nil
		}
	} else if request.MultipartBody != nil {
		// Image upload request - parse multipart form
		imageData, duration, err = parseMultipartPush(request.MultipartBody)
		if err != nil {
			return PushChannelContent400JSONResponse{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			}, nil
		}
	} else {
		return PushChannelContent400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "No content provided",
		}, nil
	}

	// Push to channel
	err = s.hub.PushToChannel(request.UUID, imageData, duration)
	if err != nil {
		return PushChannelContentdefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	return PushChannelContent200Response{}, nil
}

// ClearChannelPush handles DELETE /channels/{uuid}/push
func (s *Server) ClearChannelPush(ctx context.Context, request ClearChannelPushRequestObject) (ClearChannelPushResponseObject, error) {
	// Check if hub is available
	if s.hub == nil {
		return ClearChannelPushdefaultJSONResponse{
			Body:       Error{Code: http.StatusInternalServerError, Message: "Hub not available"},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	// Verify channel exists
	_, err := s.store.GetChannelByUUID(ctx, request.UUID)
	if err != nil {
		if ne.Is(err, ne.ChannelNotFound) {
			return ClearChannelPush404JSONResponse{
				Code:    http.StatusNotFound,
				Message: "Channel not found",
			}, nil
		}
		return ClearChannelPushdefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	// Clear the override
	err = s.hub.ClearChannelPush(request.UUID)
	if err != nil {
		return ClearChannelPushdefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	return ClearChannelPush200Response{}, nil
}

// PushDeviceContent handles POST /devices/{uuid}/push
func (s *Server) PushDeviceContent(ctx context.Context, request PushDeviceContentRequestObject) (PushDeviceContentResponseObject, error) {
	// Check if hub is available
	if s.hub == nil {
		return PushDeviceContentdefaultJSONResponse{
			Body:       Error{Code: http.StatusInternalServerError, Message: "Hub not available"},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	// Verify device exists
	_, err := s.store.GetDeviceByUUID(ctx, request.UUID)
	if err != nil {
		if ne.Is(err, ne.DeviceNotFound) {
			return PushDeviceContent404JSONResponse{
				Code:    http.StatusNotFound,
				Message: "Device not found",
			}, nil
		}
		return PushDeviceContentdefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	// Check if device is connected
	if !s.hub.IsDeviceConnected(request.UUID) {
		return PushDeviceContent409JSONResponse{
			Code:    http.StatusConflict,
			Message: "Device not currently connected",
		}, nil
	}

	var imageData []byte
	var duration time.Duration

	if request.JSONBody != nil {
		// Applet render request
		imageData, duration, err = s.renderAppletForPush(request.JSONBody)
		if err != nil {
			return PushDeviceContent400JSONResponse{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			}, nil
		}
	} else if request.MultipartBody != nil {
		// Image upload request - parse multipart form
		imageData, duration, err = parseMultipartPush(request.MultipartBody)
		if err != nil {
			return PushDeviceContent400JSONResponse{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			}, nil
		}
	} else {
		return PushDeviceContent400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "No content provided",
		}, nil
	}

	// Push to device
	err = s.hub.PushToDevice(request.UUID, imageData, duration)
	if err != nil {
		if err.Error() == "device not connected" {
			return PushDeviceContent409JSONResponse{
				Code:    http.StatusConflict,
				Message: "Device not currently connected",
			}, nil
		}
		return PushDeviceContentdefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	return PushDeviceContent200Response{}, nil
}

// ClearDevicePush handles DELETE /devices/{uuid}/push
func (s *Server) ClearDevicePush(ctx context.Context, request ClearDevicePushRequestObject) (ClearDevicePushResponseObject, error) {
	// Check if hub is available
	if s.hub == nil {
		return ClearDevicePushdefaultJSONResponse{
			Body:       Error{Code: http.StatusInternalServerError, Message: "Hub not available"},
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	// Verify device exists
	_, err := s.store.GetDeviceByUUID(ctx, request.UUID)
	if err != nil {
		if ne.Is(err, ne.DeviceNotFound) {
			return ClearDevicePush404JSONResponse{
				Code:    http.StatusNotFound,
				Message: "Device not found",
			}, nil
		}
		return ClearDevicePushdefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	// Check if device is connected
	if !s.hub.IsDeviceConnected(request.UUID) {
		return ClearDevicePush409JSONResponse{
			Code:    http.StatusConflict,
			Message: "Device not currently connected",
		}, nil
	}

	// Clear the override
	err = s.hub.ClearDevicePush(request.UUID)
	if err != nil {
		if err.Error() == "device not connected" {
			return ClearDevicePush409JSONResponse{
				Code:    http.StatusConflict,
				Message: "Device not currently connected",
			}, nil
		}
		return ClearDevicePushdefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	return ClearDevicePush200Response{}, nil
}

// parseMultipartPush parses a multipart form with image and optional duration
func parseMultipartPush(reader *multipart.Reader) ([]byte, time.Duration, error) {
	var imageData []byte
	duration := time.Duration(defaultDuration) * time.Second

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}

		switch part.FormName() {
		case "image":
			imageData, err = io.ReadAll(io.LimitReader(part, maxImageSize+1))
			if err != nil {
				return nil, 0, err
			}
		case "duration":
			data, err := io.ReadAll(part)
			if err != nil {
				return nil, 0, err
			}
			d, err := strconv.Atoi(string(data))
			if err != nil {
				return nil, 0, &validationError{"invalid duration value"}
			}
			duration = time.Duration(d) * time.Second
		}
		part.Close()
	}

	if imageData == nil {
		return nil, 0, &validationError{"no image provided"}
	}

	// Validate the image
	if err := validateWebP(imageData); err != nil {
		return nil, 0, err
	}

	return imageData, duration, nil
}

// renderAppletForPush renders an applet and returns the image data
func (s *Server) renderAppletForPush(req *PushAppletRequest) ([]byte, time.Duration, error) {
	if s.hub == nil || s.hub.Catalog == nil {
		return nil, 0, &validationError{"catalog not available"}
	}

	// Find the applet
	manifest := s.hub.Catalog.FindManifest(req.Applet)
	if manifest == nil {
		return nil, 0, &validationError{"applet not found: " + req.Applet}
	}

	// Prepare config (dereference pointer if non-nil)
	var config map[string]interface{}
	if req.Config != nil {
		config = *req.Config
	}

	// Render the applet
	imageData, err := s.hub.Catalog.RenderApplet(manifest, config)
	if err != nil {
		return nil, 0, err
	}

	// Parse duration
	duration := time.Duration(defaultDuration) * time.Second
	if req.Duration != nil {
		duration = time.Duration(*req.Duration) * time.Second
	}

	return imageData, duration, nil
}
