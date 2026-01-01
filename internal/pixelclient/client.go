package pixelclient

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/coder/websocket"
)

// Client manages a WebSocket connection to a PixelGateway server.
type Client struct {
	device *Device
	conn   *websocket.Conn
	frames chan []byte
	errors chan error
	done   chan struct{}
}

// NewClient creates a new Client for the given device.
func NewClient(device *Device) *Client {
	return &Client{
		device: device,
		frames: make(chan []byte, 1),
		errors: make(chan error, 1),
		done:   make(chan struct{}),
	}
}

// Connect establishes a WebSocket connection to the server.
func (c *Client) Connect(ctx context.Context) error {
	u := url.URL{
		Scheme:   "ws",
		Host:     c.device.Server,
		Path:     "/ws",
		RawQuery: "device=" + url.QueryEscape(c.device.ID),
	}

	conn, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	c.conn = conn

	// Start the read loop
	go c.readLoop(ctx)

	return nil
}

// readLoop reads messages from the WebSocket connection.
func (c *Client) readLoop(ctx context.Context) {
	defer close(c.done)

	for {
		msgType, data, err := c.conn.Read(ctx)
		if err != nil {
			select {
			case c.errors <- err:
			default:
			}
			return
		}

		// We only care about binary messages (WebP images)
		if msgType == websocket.MessageBinary {
			select {
			case c.frames <- data:
			default:
				// Drop frame if channel is full (keep latest)
				select {
				case <-c.frames:
				default:
				}
				c.frames <- data
			}
		}
	}
}

// Frames returns a channel that receives WebP image frames.
func (c *Client) Frames() <-chan []byte {
	return c.frames
}

// Errors returns a channel that receives connection errors.
func (c *Client) Errors() <-chan error {
	return c.errors
}

// Done returns a channel that is closed when the connection is closed.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// Close closes the WebSocket connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close(websocket.StatusNormalClosure, "client closing")
	}
	return nil
}

// Device returns the device associated with this client.
func (c *Client) Device() *Device {
	return c.device
}

// Reconnect attempts to reconnect to the server with exponential backoff.
func (c *Client) Reconnect(ctx context.Context) error {
	if c.conn != nil {
		c.conn.Close(websocket.StatusNormalClosure, "reconnecting")
		c.conn = nil
	}

	// Reset channels
	c.frames = make(chan []byte, 1)
	c.errors = make(chan error, 1)
	c.done = make(chan struct{})

	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := c.Connect(ctx); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
