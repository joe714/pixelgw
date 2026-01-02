# Push Content API Design

## Overview

This document describes the API design for pushing temporary content to channels and devices, bypassing the normal applet render loop.

## Use Cases

1. **Manual image push** - Upload a WebP image to display temporarily (e.g., notification, alert, custom graphic)
2. **On-demand applet render** - Trigger a specific applet with config to render and display immediately
3. **Device-specific override** - Push content to a single device independent of its channel

## API Endpoints

### POST /channels/{uuid}/push

Push temporary content to all devices subscribed to a channel.

**Request (Image Upload):**
```
Content-Type: multipart/form-data

image: <binary WebP data, max 128KB>
duration: <integer, seconds, optional, default: 15>
```

**Request (Applet Render):**
```
Content-Type: application/json

{
  "applet": "string",       // applet ID from catalog
  "config": {},             // applet configuration (optional)
  "duration": 30            // display time in seconds (optional, default: 15)
}
```

**Response:**
- `200 OK` - Content pushed successfully
- `400 Bad Request` - Invalid image format, size exceeded, or invalid applet
- `404 Not Found` - Channel not found

**Behavior:**
1. If image upload: validate WebP format and size (max 128KB)
2. If applet render: load applet, run with config, encode to WebP
3. Broadcast image to all currently connected devices on the channel
4. Pause the channel's render loop timer
5. After duration elapses, resume normal render loop

### POST /devices/{uuid}/push

Push temporary content to a specific device, overriding its channel subscription temporarily.

**Request:** Same as channel push (both multipart/form-data and application/json supported)

**Response:**
- `200 OK` - Content pushed successfully
- `400 Bad Request` - Invalid image format, size exceeded, or invalid applet
- `404 Not Found` - Device not found
- `409 Conflict` - Device not currently connected

**Behavior:**
1. Validate and prepare image (same as channel push)
2. Send image directly to the specific device
3. Set device "override until" timestamp
4. Device ignores channel broadcasts until:
   - Override timer expires, OR
   - New push is sent to the device, OR
   - Override is cleared via DELETE
5. When override expires, hub immediately sends the channel's last rendered image to the device (from `channel.last`), ensuring instant return to normal content without waiting for next render cycle

### DELETE /channels/{uuid}/push

Clear any active push override on a channel and resume normal render loop immediately.

**Response:**
- `200 OK` - Override cleared (or no override was active)
- `404 Not Found` - Channel not found

**Behavior:**
1. Clear channel override state
2. Reset render timer to fire immediately
3. Next render broadcasts to all devices as normal

### DELETE /devices/{uuid}/push

Clear any active push override on a device and return to channel content immediately.

**Response:**
- `200 OK` - Override cleared (or no override was active)
- `404 Not Found` - Device not found
- `409 Conflict` - Device not currently connected

**Behavior:**
1. Clear device override state
2. Immediately send the channel's last rendered image to the device
3. Device resumes receiving normal channel broadcasts

## Implementation Details

### Hub Changes

```go
// New method on Hub
func (h *Hub) PushToChannel(channelUUID uuid.UUID, image []byte, duration time.Duration) error

// New method on Hub
func (h *Hub) PushToDevice(deviceUUID uuid.UUID, image []byte, duration time.Duration) error
```

### Channel Changes

```go
// Channel needs to track override state
type Channel struct {
    // ... existing fields
    overrideUntil time.Time  // when set, pause render loop
}
```

### Client Changes

```go
// Client needs to track override state
type Client struct {
    // ... existing fields
    overrideUntil time.Time  // when set, ignore channel broadcasts
}
```

### Message Flow

**Channel Push:**
```
API Request
    │
    ▼
Hub.PushToChannel()
    │
    ├──► Channel.setOverride(duration)
    │        │
    │        ▼
    │    Pause render timer
    │
    └──► Broadcast to all channel clients
             │
             ▼
         Client.send <- image
```

**Device Push:**
```
API Request
    │
    ▼
Hub.PushToDevice()
    │
    ├──► Find client by device UUID
    │
    ├──► Client.setOverride(duration)
    │
    └──► Client.send <- image
```

### Image Validation

- Format: WebP only (check magic bytes: `RIFF....WEBP`)
- Max size: 128KB (131,072 bytes)
- Dimensions: Must be 64x32, or post will fail

### Applet Rendering

For JSON requests with applet specification:
1. Look up applet in catalog
2. Validate config against schema (if available)
3. Execute applet with config using pixlet runtime
4. Encode output to WebP
5. Proceed as image push

## OpenAPI Schema

```yaml
/channels/{uuid}/push:
  post:
    summary: Push temporary content to channel
    operationId: pushChannelContent
    parameters:
      - name: uuid
        in: path
        required: true
        schema:
          type: string
          format: uuid
    requestBody:
      required: true
      content:
        multipart/form-data:
          schema:
            type: object
            required:
              - image
            properties:
              image:
                type: string
                format: binary
                description: WebP image (max 128KB)
              duration:
                type: integer
                description: Display duration in seconds
                default: 15
        application/json:
          schema:
            type: object
            required:
              - applet
            properties:
              applet:
                type: string
                description: Applet ID from catalog
              config:
                type: object
                description: Applet configuration
                additionalProperties: true
              duration:
                type: integer
                description: Display duration in seconds
                default: 15
    responses:
      '200':
        description: Content pushed successfully
      '400':
        description: Bad request
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Error'
      '404':
        description: Channel not found
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Error'

/devices/{uuid}/push:
  post:
    summary: Push temporary content to device
    operationId: pushDeviceContent
    parameters:
      - name: uuid
        in: path
        required: true
        schema:
          type: string
          format: uuid
    requestBody:
      # Same as channel push
    responses:
      '200':
        description: Content pushed successfully
      '400':
        description: Bad request
      '404':
        description: Device not found
      '409':
        description: Device not connected
```

## Edge Cases

1. **Push to channel with no connected devices** - Succeeds silently, image is lost
2. **Push to disconnected device** - Returns 409 Conflict
3. **Rapid successive pushes** - Each push replaces the previous, resets duration timer
4. **Device reconnects during override** - Override state is lost, device resumes normal channel
5. **Channel push while device has override** - Device ignores until its override expires
6. **Duration of 0** - Display indefinitely until next push or channel render (for devices) or next scheduled render (for channels)

## Future Considerations

- **Priority levels** - Allow push to interrupt even device overrides
- **Queuing** - Queue multiple pushes instead of replacing
- **Persistence** - Store last pushed image for reconnecting devices
- **WebSocket push** - Allow pushing via WebSocket connection instead of REST
