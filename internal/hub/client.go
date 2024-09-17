package hub

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 64 * 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var lastSessionID atomic.Uint32

type ClientImage struct {
	ttl  time.Duration
	data []byte
}

// TODO naming here is not quite right.
// This is really a session or a connection, and ID is really
// the ClientUUID / connection string, which is either a device ID or
// a future extension for anonymous subscriptions directly to a channel
type Client struct {
	SessionID uint32
	UUID      uuid.UUID
	hub       atomic.Pointer[Hub]
	conn      *websocket.Conn
	send      chan *ClientImage
}

func NewClient(clientUUID uuid.UUID, conn *websocket.Conn) *Client {
	client := Client{
		SessionID: lastSessionID.Add(1),
		UUID:      clientUUID,
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
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Client %v read error: %v\n", c, err)
			} else {
				log.Printf("Client %v disconnected\n", c)
			}
			break
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
			err := c.write(msg.data)
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
