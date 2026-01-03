package hub

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait         = 10 * time.Second
	pongWait          = 60 * time.Second
	pingPeriod        = (pongWait * 9) / 10
	maxMessageSize    = 64 * 1024
	maxDeviceInfoSize = 1024 // Max size for device info payload (1KB)
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var lastSessionID atomic.Uint32

type ClientImage struct {
	TTL  time.Duration
	Data []byte
}

// TODO naming here is not quite right.
// This is really a session or a connection, and ID is really
// the ClientUUID / connection string, which is either a device ID or
// a future extension for anonymous subscriptions directly to a channel
type Client struct {
	SessionID     uint32
	UUID          uuid.UUID
	RealIP        string // The actual client IP (may differ from conn.RemoteAddr if proxied)
	Ephemeral     bool   // True for ephemeral channel clients (virtual displays)
	DisplayName   string // Human-readable name for virtual displays
	hub           atomic.Pointer[Hub]
	conn          *websocket.Conn
	send          chan *ClientImage
	overrideUntil time.Time                     // when set, ignore channel broadcasts
	OnDeviceInfo  func(uuid.UUID, string) error // callback for device info messages
}

func NewClient(clientUUID uuid.UUID, conn *websocket.Conn, realIP string) *Client {
	client := Client{
		SessionID: lastSessionID.Add(1),
		UUID:      clientUUID,
		RealIP:    realIP,
		conn:      conn,
		send:      make(chan *ClientImage, 1),
	}
	go client.writePump()
	go client.readPump()
	return &client
}

func (c *Client) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Client) start() {
	go c.writePump()
	go c.readPump()
}

func (c *Client) shutdown() {
	hub := c.hub.Swap(nil)
	if hub == nil {
		return
	}
	log.Printf("Shutdown connection %v\n", c)
	hub.unregister(c)
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
	c.conn.Close()
	close(c.send)
}

func (c *Client) readPump() {
	defer func() {
		log.Printf("%v readPump stopped\n", c)
		c.shutdown()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Client %v read error: %v\n", c, err)
			} else {
				log.Printf("Client %v disconnected\n", c)
			}
			break
		}

		// Process text messages (device info, ACKs, commands)
		if messageType == websocket.TextMessage {
			c.handleTextMessage(message)
		}
	}
}

func (c *Client) handleTextMessage(message []byte) {
	if len(message) > maxDeviceInfoSize {
		log.Printf("%v message too large (%d bytes), dropping", c, len(message))
		return
	}

	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("%v invalid JSON: %v", c, err)
		return
	}

	// Check for ACK message
	if ack, hasAck := msg["ack"]; hasAck {
		log.Printf("%v received ACK: %v", c, ack)
		return
	}

	// Check for cmd field (new format)
	if cmd, hasCmd := msg["cmd"]; hasCmd {
		switch cmd {
		case "device-info":
			// Remove cmd field before storing
			delete(msg, "cmd")
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("%v failed to re-marshal device info: %v", c, err)
				return
			}
			c.storeDeviceInfo(data)
		default:
			log.Printf("%v received unknown command: %v", c, cmd)
		}
		return
	}

	// Legacy format: check for device field (backward compatibility)
	if _, hasDevice := msg["device"]; hasDevice {
		c.storeDeviceInfo(message)
		return
	}

	log.Printf("%v received unrecognized message format", c)
}

func (c *Client) storeDeviceInfo(data []byte) {
	if c.OnDeviceInfo != nil {
		if err := c.OnDeviceInfo(c.UUID, string(data)); err != nil {
			log.Printf("%v failed to store device info: %v", c, err)
		} else {
			log.Printf("%v device info updated", c)
		}
	}
}

func (c *Client) writePump() {
	ping := time.NewTicker(pingPeriod)

	defer func() {
		log.Printf("%v writePump stopped\n", c)
		ping.Stop()
		c.shutdown()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				// Closed channel means we're already deregistered
				return
			}
			err := c.write(msg.Data)
			if err != nil {
				break
			}
		case <-ping.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				break
			}
		}
	}
}

func (c *Client) write(data []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	w, err := c.conn.NextWriter(websocket.BinaryMessage)
	if err == nil {
		l, err := w.Write(data)
		if err != nil {
			log.Printf("%v write() length: %v, error: %v", c, l, err)
		}
		err = w.Close()
		if err != nil {
			log.Printf("%v close() error: %v", c, err)
		}
	}
	return err
}

func (c *Client) String() string {
	return fmt.Sprintf("[%d %v]", c.SessionID, c.UUID)
}

// SendOTACommand sends an OTA update command to the device
func (c *Client) SendOTACommand(downloadPath string) error {
	// Build JSON command: {"cmd": "ota", "path": "/firmware/abc123"}
	cmd := map[string]string{
		"cmd":  "ota",
		"path": downloadPath,
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	err = c.conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Printf("%v failed to send OTA command: %v", c, err)
		return err
	}
	log.Printf("%v sent OTA command: %s", c, downloadPath)
	return nil
}
