package durable

import (
	"context"
	"errors"
	"time"

	"github.com/canonical/sqlair"
	"github.com/google/uuid"

	ne "github.com/joe714/pixelgw/internal/errors"
)

// Firmware represents a firmware record in the database
type Firmware struct {
	UUID           uuid.UUID `db:"uuid"`
	Platform       string    `db:"platform"`
	Filename       string    `db:"filename"`
	Description    *string   `db:"description"`
	Version        string    `db:"version"`
	BuildTimestamp string    `db:"build_timestamp"`
	ElfSHA256      string    `db:"elf_sha256"`
	IDFVersion     string    `db:"idf_version"`
	FileSize       int64     `db:"file_size"`
	IsDefault      bool      `db:"is_default"`
	UploadedAt     string    `db:"uploaded_at"`
}

// GetAllFirmwares returns all firmwares, optionally filtered by platform
func (store *Store) GetAllFirmwares(ctx context.Context, platform *string) ([]Firmware, error) {
	resp := []Firmware{}
	err := store.View(ctx, func(tx *TX) error {
		var stmt *sqlair.Statement
		if platform != nil && *platform != "" {
			stmt = sqlair.MustPrepare(
				`SELECT (uuid, platform, filename, description, version, build_timestamp, elf_sha256, idf_version, file_size, is_default, uploaded_at)
				     AS (&Firmware.uuid, &Firmware.platform, &Firmware.filename, &Firmware.description, &Firmware.version, &Firmware.build_timestamp, &Firmware.elf_sha256, &Firmware.idf_version, &Firmware.file_size, &Firmware.is_default, &Firmware.uploaded_at)
				   FROM firmwares
				  WHERE platform = $M.platform
				  ORDER BY uploaded_at DESC`,
				Firmware{},
				sqlair.M{})
			return tx.Query(stmt, sqlair.M{"platform": *platform}).GetAll(&resp)
		}
		stmt = sqlair.MustPrepare(
			`SELECT (uuid, platform, filename, description, version, build_timestamp, elf_sha256, idf_version, file_size, is_default, uploaded_at)
			     AS (&Firmware.uuid, &Firmware.platform, &Firmware.filename, &Firmware.description, &Firmware.version, &Firmware.build_timestamp, &Firmware.elf_sha256, &Firmware.idf_version, &Firmware.file_size, &Firmware.is_default, &Firmware.uploaded_at)
			   FROM firmwares
			  ORDER BY platform, uploaded_at DESC`,
			Firmware{})
		return tx.Query(stmt).GetAll(&resp)
	})
	if errors.Is(err, sqlair.ErrNoRows) {
		return resp, nil
	}
	return resp, err
}

// GetFirmwareByUUID returns a firmware by its UUID
func (store *Store) GetFirmwareByUUID(ctx context.Context, id uuid.UUID) (*Firmware, error) {
	resp := Firmware{}
	err := store.View(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare(
			`SELECT (uuid, platform, filename, description, version, build_timestamp, elf_sha256, idf_version, file_size, is_default, uploaded_at)
			     AS (&Firmware.uuid, &Firmware.platform, &Firmware.filename, &Firmware.description, &Firmware.version, &Firmware.build_timestamp, &Firmware.elf_sha256, &Firmware.idf_version, &Firmware.file_size, &Firmware.is_default, &Firmware.uploaded_at)
			   FROM firmwares
			  WHERE uuid = $M.uuid`,
			Firmware{},
			sqlair.M{})
		return tx.Query(stmt, sqlair.M{"uuid": id}).Get(&resp)
	})
	if errors.Is(err, sqlair.ErrNoRows) {
		return nil, ne.FirmwareNotFound
	}
	return &resp, err
}

// GetFirmwareByPlatformAndSHA256 checks if a firmware with the same platform and SHA256 exists
func (store *Store) GetFirmwareByPlatformAndSHA256(ctx context.Context, platform, elfSHA256 string) (*Firmware, error) {
	resp := Firmware{}
	err := store.View(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare(
			`SELECT (uuid, platform, filename, description, version, build_timestamp, elf_sha256, idf_version, file_size, is_default, uploaded_at)
			     AS (&Firmware.uuid, &Firmware.platform, &Firmware.filename, &Firmware.description, &Firmware.version, &Firmware.build_timestamp, &Firmware.elf_sha256, &Firmware.idf_version, &Firmware.file_size, &Firmware.is_default, &Firmware.uploaded_at)
			   FROM firmwares
			  WHERE platform = $M.platform AND elf_sha256 = $M.elf_sha256`,
			Firmware{},
			sqlair.M{})
		return tx.Query(stmt, sqlair.M{"platform": platform, "elf_sha256": elfSHA256}).Get(&resp)
	})
	if errors.Is(err, sqlair.ErrNoRows) {
		return nil, nil
	}
	return &resp, err
}

// CreateFirmware creates a new firmware record
func (store *Store) CreateFirmware(ctx context.Context, fw *Firmware) error {
	return store.Update(ctx, func(tx *TX) error {
		// If this firmware should be default, unset any existing default for this platform
		if fw.IsDefault {
			stmt := sqlair.MustPrepare(
				`UPDATE firmwares SET is_default = 0 WHERE platform = $M.platform AND is_default = 1`,
				sqlair.M{})
			if err := tx.Query(stmt, sqlair.M{"platform": fw.Platform}).Run(); err != nil {
				return err
			}
		}

		stmt := sqlair.MustPrepare(
			`INSERT INTO firmwares (uuid, platform, filename, description, version, build_timestamp, elf_sha256, idf_version, file_size, is_default, uploaded_at)
			 VALUES ($M.uuid, $M.platform, $M.filename, $M.description, $M.version, $M.build_timestamp, $M.elf_sha256, $M.idf_version, $M.file_size, $M.is_default, $M.uploaded_at)`,
			sqlair.M{})
		return tx.Query(stmt, sqlair.M{
			"uuid":            fw.UUID,
			"platform":        fw.Platform,
			"filename":        fw.Filename,
			"description":     fw.Description,
			"version":         fw.Version,
			"build_timestamp": fw.BuildTimestamp,
			"elf_sha256":      fw.ElfSHA256,
			"idf_version":     fw.IDFVersion,
			"file_size":       fw.FileSize,
			"is_default":      fw.IsDefault,
			"uploaded_at":     fw.UploadedAt,
		}).Run()
	})
}

// DeleteFirmware deletes a firmware record
func (store *Store) DeleteFirmware(ctx context.Context, id uuid.UUID) error {
	return store.Update(ctx, func(tx *TX) error {
		stmt := sqlair.MustPrepare(
			`DELETE FROM firmwares WHERE uuid = $M.uuid`,
			sqlair.M{})
		return tx.Query(stmt, sqlair.M{"uuid": id}).Run()
	})
}

// SetFirmwareDefault sets a firmware as the default for its platform
func (store *Store) SetFirmwareDefault(ctx context.Context, id uuid.UUID, isDefault bool) error {
	return store.Update(ctx, func(tx *TX) error {
		// Get the firmware to find its platform
		fw := Firmware{}
		stmt := sqlair.MustPrepare(
			`SELECT (uuid, platform) AS (&Firmware.uuid, &Firmware.platform) FROM firmwares WHERE uuid = $M.uuid`,
			Firmware{},
			sqlair.M{})
		if err := tx.Query(stmt, sqlair.M{"uuid": id}).Get(&fw); err != nil {
			if errors.Is(err, sqlair.ErrNoRows) {
				return ne.FirmwareNotFound
			}
			return err
		}

		if isDefault {
			// Unset any existing default for this platform
			stmt = sqlair.MustPrepare(
				`UPDATE firmwares SET is_default = 0 WHERE platform = $M.platform AND is_default = 1`,
				sqlair.M{})
			if err := tx.Query(stmt, sqlair.M{"platform": fw.Platform}).Run(); err != nil {
				return err
			}
		}

		// Set the new default
		stmt = sqlair.MustPrepare(
			`UPDATE firmwares SET is_default = $M.is_default WHERE uuid = $M.uuid`,
			sqlair.M{})
		return tx.Query(stmt, sqlair.M{"uuid": id, "is_default": isDefault}).Run()
	})
}

// GetFirmwarePath returns the file path for a firmware
func GetFirmwarePath(platform string, id uuid.UUID) string {
	return "./etc/firmwares/" + platform + "/" + id.String() + ".bin"
}

// GetFirmwareDir returns the directory path for firmware files
func GetFirmwareDir(platform string) string {
	return "./etc/firmwares/" + platform
}

// FirmwareStorageInit creates the firmware storage directories
func FirmwareStorageInit() error {
	return nil // Directory creation is handled by the API layer when needed
}

// BuildTimestampFromTime converts a time.Time to the database format (RFC3339)
func BuildTimestampFromTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
