# Firmware Update Mechanism - Implementation Specification

## Overview

Add firmware image management to PixelGW, allowing:
- Upload and storage of firmware binaries by platform
- Automatic metadata extraction (version, build date, SHA256)
- REST API for firmware management
- WebSocket-based OTA triggers to connected devices
- Time-limited download tokens for device firmware retrieval

## Architecture

### OTA Flow
```
┌─────────────┐     POST firmware      ┌─────────────┐
│  Admin UI   │ ───────────────────►   │  PixelGW    │
│             │                        │  REST API   │
└─────────────┘                        └──────┬──────┘
                                              │
                                              ▼
                                       ┌─────────────┐
                                       │  firmwares/ │
                                       │  (storage)  │
                                       └──────┬──────┘
                                              │
┌─────────────┐   WebSocket cmd:ota    ┌──────┴──────┐
│  ESP32      │ ◄──────────────────    │  PixelGW    │
│  Device     │   {path: "/fw/xyz"}    │  Hub        │
└──────┬──────┘                        └─────────────┘
       │
       │ GET /firmware/{token}
       ▼
┌─────────────┐     Binary response    ┌─────────────┐
│  ESP32      │ ◄──────────────────    │  PixelGW    │
│  OTA Client │                        │  /firmware  │
└─────────────┘                        └─────────────┘
```

**Key Design Decisions:**
- **WebSocket for OTA triggers**: Devices receive OTA commands via WebSocket text frames, not HTTP POST to device IP. This eliminates network topology concerns (NAT, VLANs, firewalls).
- **Device constructs URL**: Gateway sends only the path portion (`/firmware/{token}`). Device constructs full URL using its known gateway endpoint.
- **Fire and forget**: Gateway doesn't track OTA state. User can retry manually if needed.
- **Time-limited tokens**: Download URLs expire after 5 minutes; tokens stored in memory and invalidated on server restart.

## WebSocket Protocol Changes

### Message Format
Control messages use JSON text frames with a flat union structure for memory efficiency on ESP32:

**Server → Device (OTA command):**
```json
{"cmd": "ota", "path": "/firmware/abc123xyz"}
```

**Device → Server (ACK):**
```json
{"ack": "ota"}
```

**Device → Server (Device info - backward compatible):**
```json
{"cmd": "device-info", "device": "esp32", "sha256": "abc123...", ...}
```

### Backward Compatibility
- If incoming JSON has `device:` field but no `cmd:` field, treat as legacy device-info
- Strip `cmd:` field before storing device info
- Eventually deprecate payloads without `cmd:` field once all devices upgraded

### ACK Handling
- Gateway logs ACKs for debugging only
- No state changes or timestamp updates on ACK receipt
- Old firmware that doesn't ACK is acceptable (manual flash will update all devices first)

## Storage Design

### File-based Storage
Firmware files stored in the same Docker volume as the database:
```
/app/etc/
├── cfg.db                    # Existing SQLite database
└── firmwares/
    └── esp32/
        ├── abc123.bin        # Firmware files named by UUID
        └── def456.bin
```

### Database Schema (new table, migration v4)
```sql
CREATE TABLE firmwares (
    uuid TEXT PRIMARY KEY COLLATE NOCASE,
    platform TEXT NOT NULL,           -- "esp32", future: "esp32s3", etc.
    filename TEXT NOT NULL,           -- Original upload filename
    description TEXT,                 -- User-provided description
    version TEXT NOT NULL,            -- Extracted from binary
    build_timestamp TEXT NOT NULL,    -- ISO 8601 combined date+time
    elf_sha256 TEXT NOT NULL,         -- Extracted from binary (hex)
    idf_version TEXT NOT NULL,        -- Extracted from binary
    file_size INTEGER NOT NULL,       -- Binary size in bytes
    is_default BOOLEAN DEFAULT FALSE, -- Default for this platform
    uploaded_at TEXT NOT NULL,        -- RFC3339 timestamp
    UNIQUE (platform, elf_sha256)     -- Prevent duplicate uploads
);

CREATE INDEX idx_firmwares_platform ON firmwares (platform, is_default);
```

### Duplicate Upload Handling
- Upload of binary with matching `(platform, elf_sha256)` returns **409 Conflict**
- User must delete existing firmware first if they want to re-upload with different metadata

### Default Firmware Logic
- Setting `is_default=true` automatically clears previous default for that platform
- Only one default per platform

## Metadata Extraction

### ESP32 Binary Format
Per `PixelFirmware/docs/firmware-binary-format.md`:
- `esp_app_desc_t` structure at offset 0x20 (256 bytes)
- Magic word: 0xABCD5432 at offset 0x20
- Version: offset 0x30, 32 bytes (null-terminated)
- Build time: offset 0x70, 16 bytes (null-terminated, e.g., "04:52:44")
- Build date: offset 0x80, 16 bytes (null-terminated, e.g., "Jan  3 2026")
- IDF version: offset 0x90, 32 bytes (null-terminated)
- ELF SHA256: offset 0xB0, 32 bytes (binary)

### Go Implementation
```go
// internal/firmware/esp32.go
type ESP32Metadata struct {
    Version        string    // e.g., "b62bc47"
    BuildTimestamp time.Time // Combined and parsed to ISO 8601
    IDFVersion     string    // e.g., "v5.2.6-510-gef0c4d3b8d"
    ElfSHA256      string    // hex encoded, 64 chars
}

func ExtractESP32Metadata(data []byte) (*ESP32Metadata, error) {
    // Validate minimum size and magic word
    // Parse build date + build time into single time.Time
    // Return error if magic word invalid (rejects non-ESP32 binaries)
}
```

### Validation Rules
- **Maximum file size**: 2MB (enforced in Go handler via `http.MaxBytesReader`)
- **Magic word validation**: Must be 0xABCD5432 at offset 0x20
- **Invalid binaries rejected**: 400 Bad Request if metadata extraction fails

## Download Token System

### Design
- Separate endpoint: `GET /firmware/{token}` (not under `/api/`)
- Token is random string mapped to `{firmwareUUID, expiresAt}` in memory
- **Expiry**: 5 minutes from token creation
- **Invalidation**: All tokens invalidated on server restart (acceptable for single-tenant system)
- **No signing/HMAC**: Simple memory-based lookup

### Token Storage
```go
type downloadToken struct {
    FirmwareUUID string
    ExpiresAt    time.Time
}

var activeTokens = sync.Map{} // token string -> downloadToken
```

### Token Generation
When OTA is triggered:
1. Generate random token (e.g., 32-char hex string)
2. Store mapping in memory with 5-minute expiry
3. Send path `/firmware/{token}` to device via WebSocket

## REST API Design

### Endpoints

#### List Firmwares
```
GET /api/firmwares
GET /api/firmwares?platform=esp32
```
Response: Array of FirmwareSummary objects

#### Upload Firmware
```
POST /api/firmwares
Content-Type: multipart/form-data

platform: esp32
description: "v1.0 stable release"
is_default: true
firmware: <binary file, max 2MB>
```
Response:
- **201 Created**: FirmwareSummary of uploaded firmware
- **400 Bad Request**: Invalid binary (wrong magic, extraction failed)
- **409 Conflict**: Duplicate binary already exists for this platform

#### Get Firmware Details
```
GET /api/firmwares/{uuid}
```

#### Delete Firmware
```
DELETE /api/firmwares/{uuid}
```
- Deletion allowed even if devices are running this firmware
- No warnings or blocks

#### Set Default Firmware
```
PATCH /api/firmwares/{uuid}
Content-Type: application/json

{"is_default": true}
```
Automatically unsets previous default for the platform.

#### Trigger Device OTA
```
POST /api/devices/{uuid}/ota
Content-Type: application/json

{"firmware_uuid": "abc-123"}
```
Response:
- **200 OK**: OTA command sent, includes `{"path": "/firmware/{token}"}`
- **404 Not Found**: Device or firmware UUID not found
- **409 Conflict**: Device is not connected (offline)

#### Download Firmware (Token-based)
```
GET /firmware/{token}
```
Response:
- **200 OK**: Binary file with `Content-Type: application/octet-stream`
- **404 Not Found**: Token invalid or expired

### OpenAPI Schema Additions

```yaml
paths:
  /firmwares:
    get:
      operationId: getFirmwares
      parameters:
        - name: platform
          in: query
          schema:
            type: string
      responses:
        '200':
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/FirmwareSummary'
    post:
      operationId: uploadFirmware
      requestBody:
        content:
          multipart/form-data:
            schema:
              type: object
              required: [platform, firmware]
              properties:
                platform:
                  type: string
                description:
                  type: string
                is_default:
                  type: boolean
                firmware:
                  type: string
                  format: binary
      responses:
        '201':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/FirmwareSummary'
        '400':
          description: Invalid firmware binary
        '409':
          description: Duplicate firmware already exists

  /firmwares/{uuid}:
    get:
      operationId: getFirmwareByUUID
      parameters:
        - name: uuid
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/FirmwareSummary'
        '404':
          description: Firmware not found
    delete:
      operationId: deleteFirmware
      parameters:
        - name: uuid
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '204':
          description: Firmware deleted
        '404':
          description: Firmware not found
    patch:
      operationId: patchFirmware
      parameters:
        - name: uuid
          in: path
          required: true
          schema:
            type: string
            format: uuid
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                is_default:
                  type: boolean
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/FirmwareSummary'
        '404':
          description: Firmware not found

  /devices/{uuid}/ota:
    post:
      operationId: triggerDeviceOTA
      parameters:
        - name: uuid
          in: path
          required: true
          schema:
            type: string
            format: uuid
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [firmware_uuid]
              properties:
                firmware_uuid:
                  type: string
                  format: uuid
      responses:
        '200':
          content:
            application/json:
              schema:
                type: object
                properties:
                  path:
                    type: string
                    description: Download path sent to device
        '404':
          description: Device or firmware not found
        '409':
          description: Device is offline

components:
  schemas:
    FirmwareSummary:
      type: object
      required:
        - uuid
        - platform
        - filename
        - version
        - build_timestamp
        - elf_sha256
        - idf_version
        - file_size
        - is_default
        - uploaded_at
      properties:
        uuid:
          type: string
          format: uuid
        platform:
          type: string
        filename:
          type: string
        description:
          type: string
        version:
          type: string
        build_timestamp:
          type: string
          format: date-time
          description: ISO 8601 timestamp of firmware build
        elf_sha256:
          type: string
          description: Hex-encoded SHA256 of ELF file
        idf_version:
          type: string
        file_size:
          type: integer
        is_default:
          type: boolean
        uploaded_at:
          type: string
          format: date-time
```

## UI Design

### Navigation
Add "Firmwares" as top-level nav item in main sidebar alongside Channels, Devices.

### Firmware Management Page (`/firmwares`)
```
┌─────────────────────────────────────────────────────────────────┐
│ Firmwares                                    [+ Upload Firmware]│
├─────────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ esp32                                                       │ │
│ │ ┌─────────────────────────────────────────────────────────┐ │ │
│ │ │ PixelClient.bin              ★ default     🗑 Delete    │ │ │
│ │ │ Version: c8f921a                                        │ │ │
│ │ │ Built: 2026-01-05T10:30:22Z | IDF v5.2.6 | 1.2 MB       │ │ │
│ │ │ "Latest stable with new animation features"             │ │ │
│ │ └─────────────────────────────────────────────────────────┘ │ │
│ │ ┌─────────────────────────────────────────────────────────┐ │ │
│ │ │ PixelClient.bin                            🗑 Delete    │ │ │
│ │ │ Version: b62bc47                                        │ │ │
│ │ │ Built: 2026-01-03T04:52:44Z | IDF v5.2.6 | 1.1 MB       │ │ │
│ │ │ "Initial release"                                       │ │ │
│ │ └─────────────────────────────────────────────────────────┘ │ │
│ └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

**Design Notes:**
- Show platform grouping headers even with only esp32 (future-proof)
- List firmwares newest first within each platform
- Delete button inline, no confirmation dialog needed

### Upload Modal
```
┌─────────────────────────────────────────────────────────────────┐
│ Upload Firmware                                              X  │
├─────────────────────────────────────────────────────────────────┤
│ Platform:    [esp32                         ▼]                  │
│ File:        [Choose file...] PixelClient.bin                   │
│ Description: [Latest stable release                      ]      │
│ ☑ Set as default for this platform                              │
│                                                                 │
│ ── Preview (after file selected) ──                             │
│ Version: c8f921a                                                │
│ Built: 2026-01-05T10:30:22Z                                     │
│ IDF: v5.2.6-510-gef0c4d3b8d                                     │
│ SHA256: abc123def456...                                         │
│ Size: 1,234,567 bytes                                           │
├─────────────────────────────────────────────────────────────────┤
│                                             [Cancel] [Upload]   │
└─────────────────────────────────────────────────────────────────┘
```

**Client-side Metadata Extraction:**
- Read only first 512 bytes of file using `File.slice()` for preview
- Parse ESP32 header in JavaScript (magic word validation, version, dates, SHA256)
- Server validates independently on upload

### Device Config Modal Enhancement
Add "Firmware Update" section to existing `DeviceConfigModal.tsx`:

```
┌─────────────────────────────────────────────────────────────────┐
│ Configure Device                                             X  │
├─────────────────────────────────────────────────────────────────┤
│ Device ID: [550e8400-e29b-41d4...]                    [📋 Copy] │
│ Name:      [Living Room Display          ]                      │
│ Channel:   [default                        ▼]                   │
│                                                                 │
│ ▼ Device Info                             (updated Jan 3, 2026) │
│    version: b62bc47                                             │
│    sha256:  abc123...                                           │
│    compiled: 2026-01-03T04:52:44Z                               │
│                                                                 │
│ ▼ Firmware Update                                               │
│    Current: b62bc47 (2026-01-03)                                │
│    Available: [Select firmware...          ▼]                   │
│                ├─ c8f921a (2026-01-05) ★ default                │
│                ├─ b62bc47 (2026-01-03) ◀ current                │
│                └─ a1b2c3d (2025-12-28)                          │
│    [🔄 Update Firmware]                                         │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [🗑 Delete] [👁 Identify]                    [Cancel] [Save]    │
└─────────────────────────────────────────────────────────────────┘
```

**Visibility Logic:**
- Only show "Firmware Update" section if:
  1. Device is connected (`device.connected === true`)
  2. At least one firmware exists for esp32 platform
- **Update button disabled** if device is offline
- Button always enabled otherwise (even if already on default, user may want to force update)

**Dropdown Behavior:**
- Shows all firmwares for esp32 platform
- Indicates `★ default` and `◀ current` (matching elf_sha256)
- If device firmware not in system (was deleted), just show version string without match indicator

**No special OTA state:** Device shows normal disconnected state during OTA reboot. User waits for reconnection.

## Implementation Phases

### Phase 1: Backend Foundation
1. Create `internal/firmware/` package:
   - `esp32.go` - ESP32 metadata extraction with magic word validation
   - `firmware.go` - Platform-agnostic interfaces
2. Add `internal/durable/firmware.go` - Database CRUD operations
3. Add schema migration v4 in `store.go`
4. Create firmware storage directory on startup

### Phase 2: REST API
1. Update `pixelgw.yaml` with firmware endpoints
2. Run `make generate` to generate server stubs
3. Implement `internal/api/firmwares.go`:
   - Upload with 2MB limit, metadata extraction, duplicate detection
   - List, get, delete, patch operations
   - Auto-unset previous default on is_default=true
4. Add download token management (memory-based, 5-min expiry)
5. Add `/firmware/{token}` handler (outside API router)
6. Run `npm run codegen` in web/ to update TypeScript client

### Phase 3: WebSocket OTA Integration
1. Add `cmd: ota` message type handling in hub
2. Implement OTA trigger in `POST /devices/{uuid}/ota`:
   - Verify device connected
   - Generate download token
   - Send `{cmd: "ota", path: "/firmware/{token}"}` via WebSocket
3. Add ACK logging for `{"ack": "ota"}` messages
4. Update device-info parsing for backward compatibility:
   - Accept both `{cmd: "device-info", ...}` and legacy `{device: ...}` format
   - Strip `cmd` field before storing

### Phase 4: Frontend
1. Create `web/src/pages/firmwares.tsx` - Management page with platform grouping
2. Create `web/src/components/UploadFirmwareModal.tsx`:
   - Client-side partial file read (512 bytes) for preview
   - Platform dropdown, description field, is_default checkbox
3. Add firmware section to `DeviceConfigModal.tsx`:
   - Dropdown with current/default indicators
   - Update button (disabled when offline)
4. Add route `/firmwares` to `main.tsx`
5. Add "Firmwares" nav link to sidebar

### Phase 5: Firmware Changes (Pre-requisite)
Before deploying gateway changes, manually flash all devices with updated firmware that:
1. Supports `{cmd: "ota", path: "..."}` WebSocket messages
2. Sends `{ack: "ota"}` response
3. Constructs full URL from path + known gateway endpoint
4. Sends `{cmd: "device-info", ...}` on connect

### Phase 6: Testing & Polish
1. Test upload flow with real ESP32 binaries
2. Test OTA trigger and device update cycle
3. Verify token expiry (wait 5 minutes, confirm 404)
4. Test edge cases: offline device, duplicate upload, invalid binary
5. Add loading states and error messages in UI

## File Changes Summary

| File | Change |
|------|--------|
| `pixelgw.yaml` | Add firmware endpoints and schemas |
| `internal/firmware/esp32.go` | New: ESP32 metadata extraction |
| `internal/firmware/firmware.go` | New: Platform interface |
| `internal/firmware/tokens.go` | New: Download token management |
| `internal/durable/store.go` | Add migration v4 |
| `internal/durable/firmware.go` | New: Database operations |
| `internal/api/firmwares.go` | New: API handlers |
| `internal/api/download.go` | New: Token-based download handler |
| `internal/hub/hub.go` | Add OTA message routing |
| `internal/hub/client.go` | Handle OTA command sending, ACK logging |
| `web/src/pages/firmwares.tsx` | New: Management page |
| `web/src/components/UploadFirmwareModal.tsx` | New: Upload modal |
| `web/src/components/DeviceConfigModal.tsx` | Add firmware section |
| `web/src/lib/esp32-parser.ts` | New: Client-side binary parsing |
| `web/src/main.tsx` | Add route |
| `web/src/components/Sidebar.tsx` | Add nav link (or equivalent) |

## Security Considerations

- **No authentication**: Consistent with existing single-tenant design
- **Token expiry**: 5-minute window limits exposure of download URLs
- **Server restart invalidation**: Acceptable tradeoff for simplicity
- **Future improvement**: Device authentication at HTTP layer would supersede token system
