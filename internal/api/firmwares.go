package api

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/joe714/pixelgw/internal/durable"
	ne "github.com/joe714/pixelgw/internal/errors"
	"github.com/joe714/pixelgw/internal/firmware"
)

// firmwareToSummary converts a durable.Firmware to api.FirmwareSummary
func firmwareToSummary(fw *durable.Firmware) FirmwareSummary {
	buildTimestamp, _ := time.Parse(time.RFC3339, fw.BuildTimestamp)
	uploadedAt, _ := time.Parse(time.RFC3339, fw.UploadedAt)

	return FirmwareSummary{
		UUID:           fw.UUID,
		Platform:       fw.Platform,
		Filename:       fw.Filename,
		Description:    fw.Description,
		Version:        fw.Version,
		BuildTimestamp: buildTimestamp,
		ElfSHA256:      fw.ElfSHA256,
		IDFVersion:     fw.IDFVersion,
		FileSize:       fw.FileSize,
		IsDefault:      fw.IsDefault,
		UploadedAt:     uploadedAt,
	}
}

func (s *Server) GetFirmwares(ctx context.Context, request GetFirmwaresRequestObject) (GetFirmwaresResponseObject, error) {
	firmwares, err := s.store.GetAllFirmwares(ctx, request.Params.Platform)
	if err != nil {
		return GetFirmwaresdefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	resp := make([]FirmwareSummary, 0, len(firmwares))
	for _, fw := range firmwares {
		resp = append(resp, firmwareToSummary(&fw))
	}

	return GetFirmwares200JSONResponse(resp), nil
}

func (s *Server) GetFirmwareByUUID(ctx context.Context, request GetFirmwareByUUIDRequestObject) (GetFirmwareByUUIDResponseObject, error) {
	fw, err := s.store.GetFirmwareByUUID(ctx, request.UUID)
	if err != nil {
		return GetFirmwareByUUIDdefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	return GetFirmwareByUUID200JSONResponse(firmwareToSummary(fw)), nil
}

func (s *Server) UploadFirmware(ctx context.Context, request UploadFirmwareRequestObject) (UploadFirmwareResponseObject, error) {
	// Parse multipart form
	var platform string
	var description *string
	var isDefault bool
	var fileData []byte
	var filename string

	for {
		part, err := request.Body.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return UploadFirmware400JSONResponse{
				Code:    http.StatusBadRequest,
				Message: "Failed to parse multipart form: " + err.Error(),
			}, nil
		}

		switch part.FormName() {
		case "platform":
			data, _ := io.ReadAll(part)
			platform = string(data)
		case "description":
			data, _ := io.ReadAll(part)
			desc := string(data)
			if desc != "" {
				description = &desc
			}
		case "is_default":
			data, _ := io.ReadAll(part)
			isDefault = string(data) == "true"
		case "firmware":
			filename = part.FileName()
			// Read with size limit
			limitedReader := io.LimitReader(part, firmware.MaxFirmwareSize+1)
			data, err := io.ReadAll(limitedReader)
			if err != nil {
				return UploadFirmware400JSONResponse{
					Code:    http.StatusBadRequest,
					Message: "Failed to read firmware file: " + err.Error(),
				}, nil
			}
			if len(data) > firmware.MaxFirmwareSize {
				return UploadFirmware400JSONResponse{
					Code:    http.StatusBadRequest,
					Message: "Firmware file exceeds maximum size of 2MB",
				}, nil
			}
			fileData = data
		}
	}

	// Validate required fields
	if platform == "" {
		return UploadFirmware400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "Platform is required",
		}, nil
	}
	if len(fileData) == 0 {
		return UploadFirmware400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "Firmware file is required",
		}, nil
	}

	// Extract and validate metadata
	meta, err := firmware.ValidateFirmware(fileData, platform)
	if err != nil {
		return UploadFirmware400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid firmware binary: " + err.Error(),
		}, nil
	}

	// Check for duplicate
	existing, err := s.store.GetFirmwareByPlatformAndSHA256(ctx, platform, meta.ElfSHA256)
	if err != nil {
		return UploadFirmwaredefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}
	if existing != nil {
		return UploadFirmware409JSONResponse{
			Code:    http.StatusConflict,
			Message: "Firmware with same SHA256 already exists for this platform",
		}, nil
	}

	// Create firmware record
	fwUUID := uuid.New()
	now := time.Now().UTC().Format(time.RFC3339)

	fw := &durable.Firmware{
		UUID:           fwUUID,
		Platform:       platform,
		Filename:       filename,
		Description:    description,
		Version:        meta.Version,
		BuildTimestamp: durable.BuildTimestampFromTime(meta.BuildTimestamp),
		ElfSHA256:      meta.ElfSHA256,
		IDFVersion:     meta.IDFVersion,
		FileSize:       int64(len(fileData)),
		IsDefault:      isDefault,
		UploadedAt:     now,
	}

	// Ensure directory exists
	dir := durable.GetFirmwareDir(platform)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Failed to create firmware directory: %v", err)
		return UploadFirmwaredefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	// Write firmware file
	filePath := durable.GetFirmwarePath(platform, fwUUID)
	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		log.Printf("Failed to write firmware file: %v", err)
		return UploadFirmwaredefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	// Create database record
	if err := s.store.CreateFirmware(ctx, fw); err != nil {
		// Clean up file on failure
		os.Remove(filePath)
		return UploadFirmwaredefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	log.Printf("Firmware uploaded: %s %s (%s)", platform, meta.Version, fwUUID)
	return UploadFirmware201JSONResponse(firmwareToSummary(fw)), nil
}

func (s *Server) DeleteFirmware(ctx context.Context, request DeleteFirmwareRequestObject) (DeleteFirmwareResponseObject, error) {
	// Get firmware to find the file path
	fw, err := s.store.GetFirmwareByUUID(ctx, request.UUID)
	if err != nil {
		return DeleteFirmwaredefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	// Delete database record
	if err := s.store.DeleteFirmware(ctx, request.UUID); err != nil {
		return DeleteFirmwaredefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	// Delete firmware file (best effort)
	filePath := durable.GetFirmwarePath(fw.Platform, fw.UUID)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to delete firmware file %s: %v", filePath, err)
	}

	log.Printf("Firmware deleted: %s %s (%s)", fw.Platform, fw.Version, fw.UUID)
	return DeleteFirmware204Response{}, nil
}

func (s *Server) PatchFirmware(ctx context.Context, request PatchFirmwareRequestObject) (PatchFirmwareResponseObject, error) {
	if request.Body == nil || request.Body.IsDefault == nil {
		return PatchFirmwaredefaultJSONResponse{
			Body: Error{
				Code:    http.StatusBadRequest,
				Message: "No attributes provided",
			},
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	if err := s.store.SetFirmwareDefault(ctx, request.UUID, *request.Body.IsDefault); err != nil {
		return PatchFirmwaredefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	// Get updated firmware
	fw, err := s.store.GetFirmwareByUUID(ctx, request.UUID)
	if err != nil {
		return PatchFirmwaredefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	return PatchFirmware200JSONResponse(firmwareToSummary(fw)), nil
}

func (s *Server) TriggerDeviceOTA(ctx context.Context, request TriggerDeviceOTARequestObject) (TriggerDeviceOTAResponseObject, error) {
	// Verify device exists
	_, err := s.store.GetDeviceByUUID(ctx, request.UUID)
	if err != nil {
		return TriggerDeviceOTAdefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	// Verify device is connected
	if s.hub == nil || !s.hub.IsDeviceConnected(request.UUID) {
		return TriggerDeviceOTA409JSONResponse{
			Code:    http.StatusConflict,
			Message: "Device is not connected",
		}, nil
	}

	// Verify firmware exists
	fw, err := s.store.GetFirmwareByUUID(ctx, request.Body.FirmwareUUID)
	if err != nil {
		return TriggerDeviceOTAdefaultJSONResponse{
			Body:       RenderError(ne.FirmwareNotFound),
			StatusCode: StatusCode(ne.FirmwareNotFound),
		}, nil
	}

	// Generate download token
	token, err := s.tokenStore.Generate(fw.UUID)
	if err != nil {
		log.Printf("Failed to generate download token: %v", err)
		return TriggerDeviceOTAdefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	// Build download path
	downloadPath := "/firmware/" + token

	// Send OTA command to device via WebSocket
	if err := s.hub.SendOTACommand(request.UUID, downloadPath); err != nil {
		s.tokenStore.Revoke(token)
		return TriggerDeviceOTAdefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	log.Printf("OTA triggered for device %s -> firmware %s (%s)", request.UUID, fw.Version, fw.UUID)

	return TriggerDeviceOTA200JSONResponse{
		Path: &downloadPath,
	}, nil
}

// FirmwareDownloadHandler handles token-based firmware downloads
func (s *Server) FirmwareDownloadHandler(w http.ResponseWriter, r *http.Request) {
	// Extract token from path: /firmware/{token}
	token := filepath.Base(r.URL.Path)
	if token == "" || token == "firmware" {
		http.NotFound(w, r)
		return
	}

	// Validate token
	fwUUID, valid := s.tokenStore.Validate(token)
	if !valid {
		http.NotFound(w, r)
		return
	}

	// Get firmware metadata
	fw, err := s.store.GetFirmwareByUUID(r.Context(), fwUUID)
	if err != nil {
		log.Printf("Firmware download: firmware not found %s", fwUUID)
		http.NotFound(w, r)
		return
	}

	// Open and serve firmware file
	filePath := durable.GetFirmwarePath(fw.Platform, fw.UUID)
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Firmware download: failed to open %s: %v", filePath, err)
		http.Error(w, "Firmware file not found", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+fw.Filename)

	// Serve file
	http.ServeContent(w, r, fw.Filename, time.Time{}, file)
	log.Printf("Firmware download: served %s to %s", fw.Filename, r.RemoteAddr)
}
