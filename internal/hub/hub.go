package hub

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/joe714/pixelgw/internal/catalog"
	"github.com/joe714/pixelgw/internal/durable"
	ne "github.com/joe714/pixelgw/internal/errors"
)

type SessionInfo struct {
	SessionID   uint32
	DeviceUUID  uuid.UUID
	RemoteAddr  string
	ChannelUUID uuid.UUID
	ChannelName string
	Ephemeral   bool
	DisplayName string
}

type Hub struct {
	Catalog  *catalog.Catalog
	store    *durable.Store
	clients  map[*Client]*Channel
	channels map[uuid.UUID]*Channel
	tasks    chan *task
}

func NewHub(store *durable.Store) *Hub {
	hub := &Hub{
		Catalog:  catalog.NewCatalog(os.DirFS("apps")),
		store:    store,
		clients:  make(map[*Client]*Channel),
		channels: make(map[uuid.UUID]*Channel),
		tasks:    make(chan *task),
	}

	go hub.run()

	return hub
}

func (h *Hub) run() {
	for {
		select {
		case task := <-h.tasks:
			task.run()
		}
	}
}

func (h *Hub) appletsFromConfig(cfg *durable.Channel) ([]AppConfig, error) {
	apps := make([]AppConfig, 0, len(cfg.Applets))
	for _, app := range cfg.Applets {
		m := h.Catalog.FindManifest(app.AppID)
		if m == nil {
			log.Printf("%v Cannot find Applet with ID %v", cfg.Name, app.AppID)
			continue
		}
		args := make(map[string]string)
		if app.Config != nil {
			err := json.Unmarshal([]byte(*app.Config), &args)
			if err != nil {
				log.Printf(`%v Cannot unmarshal config for applet %v at index %v "%v": %v`,
					cfg.Name,
					app.AppID,
					app.Idx,
					*app.Config,
					err)
				continue
			}
		}
		apps = append(apps, AppConfig{
			UUID:     app.UUID,
			Manifest: m,
			Config:   args,
			Ttl:      0,
		})
	}
	return apps, nil
}

func (h *Hub) getChannel(channelUUID uuid.UUID) (*Channel, error) {
	ch := h.channels[channelUUID]
	if ch != nil {
		return ch, nil
	}
	cfg, err := h.store.GetChannelByUUID(context.Background(), channelUUID)
	if err != nil {
		return nil, err
	}

	apps, err := h.appletsFromConfig(cfg)
	ch = NewChannel(h, cfg.UUID, cfg.Name, apps)
	h.channels[cfg.UUID] = ch
	ch.start()
	return ch, nil
}

func (h *Hub) register(client *Client, channelUUID uuid.UUID) error {
	claimed := client.hub.CompareAndSwap(nil, h)
	if !claimed {
		return errors.New("Client registered to different hub")
	}

	err := RunTask(h.tasks, func() error {
		nxt, err := h.getChannel(channelUUID)

		if err != nil {
			return err
		}
		log.Printf("%v register %v\n",
			client,
			nxt.Name)
		nxt.subscribe(client)
		h.clients[client] = nxt
		return nil
	})
	return err
}

func (h *Hub) unregister(client *Client) {
	_ = RunTask(h.tasks, func() error {
		ch := h.clients[client]
		if ch != nil {
			log.Printf("%v deregister %v\n", client, ch.Name)
			ch.unsubscribe(client)
			delete(h.clients, client)
			// Record disconnect time in database (skip for ephemeral clients)
			if !client.Ephemeral {
				if err := h.store.LogoutDevice(context.Background(), client.UUID); err != nil {
					log.Printf("%v failed to record logout: %v\n", client, err)
				}
			}
		}
		return nil
	})
}

func (h *Hub) ReloadApplets(channelUUID uuid.UUID, first uuid.UUID) error {
	err := RunTask(h.tasks, func() error {
		ch := h.channels[channelUUID]
		if ch == nil {
			// Channel isn't currently running
			return nil
		}

		cfg, err := h.store.GetChannelByUUID(context.Background(), channelUUID)
		if err != nil {
			return err
		}
		apps, err := h.appletsFromConfig(cfg)

		log.Printf("Reload channel %v with %d applets", ch.Name, len(apps))
		if err != nil {
			return err
		}
		return ch.setApplets(apps, first)
	})
	return err
}

func (h *Hub) SubscribeDevice(deviceUUID uuid.UUID, channelUUID uuid.UUID) error {
	err := RunTask(h.tasks, func() error {
		var nxt *Channel
		for cl, ch := range h.clients {
			if cl.UUID != deviceUUID || ch.UUID == channelUUID {
				continue
			}
			if nxt == nil {
				tmp, err := h.getChannel(channelUUID)
				if err != nil {
					return err
				}
				nxt = tmp
			}
			ch.unsubscribe(cl)
			nxt.subscribe(cl)
			h.clients[cl] = nxt
		}
		return nil
	})
	return err
}

//func (h *Hub) uploadHandler(w http.ResponseWriter, r *http.Request) {
//	host, _, _ := net.SplitHostPort(r.RemoteAddr)
//	r.ParseMultipartForm(256 * 1024)
//	file, _, err := r.FormFile("image")
//	if err != nil {
//		log.Printf("%v: upload failed: %v", host, err)
//		return
//	}
//
//	defer file.Close()
//	deviceUUID := r.FormValue("device")
//	data, err := ioutil.ReadAll(file)
//	log.Printf("%v: upload for %v %v", host, deviceUUID, len(data))
//	if err != nil {
//		log.Printf("%v upload read failed: %v", host, err)
//		return
//	}
//
//	msg := &BroadcastMsg{deviceUUID: deviceUUID, ttl: 15 * time.Second, data: data}
//	h.broadcast <- msg
//}

func (h *Hub) wsHandler(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	// Check for X-Forwarded-For header (set by reverse proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first (original client)
		if idx := strings.Index(xff, ","); idx != -1 {
			host = strings.TrimSpace(xff[:idx])
		} else {
			host = strings.TrimSpace(xff)
		}
		// Strip IPv6 prefix if present (e.g., "::ffff:192.168.1.1" -> "192.168.1.1")
		host = strings.TrimPrefix(host, "::ffff:")
	}
	q := r.URL.Query()
	id := q.Get("device")

	if len(id) == 0 {
		log.Printf("%v: No device UUID specified", host)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	deviceUUID, err := uuid.Parse(id)
	if err != nil {
		log.Printf("%v %v: Device UUID is not valid: %v\n", id, host, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	device, err := h.store.LoginDevice(r.Context(), deviceUUID, host)
	if err != nil {
		log.Printf("%v %v: failed to get device configuration: %v\n", deviceUUID, host, err)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("%v %v: failed to establish websocket: %v\n", deviceUUID, host, err)
		return
	}
	client := NewClient(deviceUUID, conn, host)
	client.OnDeviceInfo = func(uuid uuid.UUID, info string) error {
		return h.store.UpdateDeviceInfo(context.Background(), uuid, info)
	}
	log.Printf("%v established from %v", client, host)
	_ = h.register(client, device.ChannelUUID)
}

func (h *Hub) GetWsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { h.wsHandler(w, r) }
}

// registerEphemeral registers an ephemeral client without device database operations
func (h *Hub) registerEphemeral(client *Client, channelUUID uuid.UUID) error {
	claimed := client.hub.CompareAndSwap(nil, h)
	if !claimed {
		return errors.New("Client registered to different hub")
	}

	err := RunTask(h.tasks, func() error {
		nxt, err := h.getChannel(channelUUID)
		if err != nil {
			return err
		}
		log.Printf("%v ephemeral register %v\n", client, nxt.Name)
		nxt.subscribe(client)
		h.clients[client] = nxt
		return nil
	})
	return err
}

// virtualDisplayHandler handles WebSocket connections for ephemeral virtual displays
func (h *Hub) virtualDisplayHandler(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	// Check for X-Forwarded-For header (set by reverse proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			host = strings.TrimSpace(xff[:idx])
		} else {
			host = strings.TrimSpace(xff)
		}
		host = strings.TrimPrefix(host, "::ffff:")
	}

	q := r.URL.Query()
	channelID := q.Get("channel")

	if len(channelID) == 0 {
		log.Printf("%v: No channel UUID specified for virtual display", host)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	channelUUID, err := uuid.Parse(channelID)
	if err != nil {
		log.Printf("%v %v: Channel UUID is not valid: %v\n", channelID, host, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify channel exists
	_, err = h.store.GetChannelByUUID(r.Context(), channelUUID)
	if err != nil {
		log.Printf("%v %v: failed to get channel: %v\n", channelUUID, host, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("%v %v: failed to establish websocket: %v\n", channelUUID, host, err)
		return
	}

	// Generate ephemeral session UUID (not persisted)
	sessionUUID := uuid.New()

	client := NewClient(sessionUUID, conn, host)
	client.Ephemeral = true
	client.DisplayName = "Virtual Display"

	log.Printf("%v virtual display established from %v for channel %v", client, host, channelUUID)
	_ = h.registerEphemeral(client, channelUUID)
}

// GetVirtualDisplayHandler returns the HTTP handler for virtual display WebSocket connections
func (h *Hub) GetVirtualDisplayHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { h.virtualDisplayHandler(w, r) }
}

func (h *Hub) GetSessions() []SessionInfo {
	resp := []SessionInfo{}
	_ = RunTask(h.tasks, func() error {
		for k, v := range h.clients {
			resp = append(resp, SessionInfo{
				SessionID:   k.SessionID,
				DeviceUUID:  k.UUID,
				RemoteAddr:  k.RealIP,
				ChannelUUID: v.UUID,
				ChannelName: v.Name,
				Ephemeral:   k.Ephemeral,
				DisplayName: k.DisplayName,
			})
		}
		return nil
	})
	return resp
}

func (h *Hub) GetLastImage(channelUUID uuid.UUID) (*ClientImage, error) {
	var resp *ClientImage
	err := RunTask(h.tasks, func() error {
		ch, err := h.getChannel(channelUUID)
		if err != nil {
			return err
		}
		resp = ch.last
		return nil
	})
	return resp, err
}

// PushToChannel pushes an image to all devices subscribed to a channel
func (h *Hub) PushToChannel(channelUUID uuid.UUID, image []byte, duration time.Duration) error {
	return RunTask(h.tasks, func() error {
		ch, err := h.getChannel(channelUUID)
		if err != nil {
			return err
		}
		return ch.pushContent(image, duration)
	})
}

// ClearChannelPush clears any active push override on a channel
func (h *Hub) ClearChannelPush(channelUUID uuid.UUID) error {
	return RunTask(h.tasks, func() error {
		ch := h.channels[channelUUID]
		if ch == nil {
			// Channel not running, nothing to clear
			return nil
		}
		return ch.clearOverride()
	})
}

// findClientByDevice looks up a client by device UUID (must be called within RunTask)
func (h *Hub) findClientByDevice(deviceUUID uuid.UUID) (*Client, *Channel) {
	for c, ch := range h.clients {
		if c.UUID == deviceUUID {
			return c, ch
		}
	}
	return nil, nil
}

// PushToDevice pushes an image to a specific device, overriding its channel subscription
func (h *Hub) PushToDevice(deviceUUID uuid.UUID, image []byte, duration time.Duration) error {
	return RunTask(h.tasks, func() error {
		client, _ := h.findClientByDevice(deviceUUID)
		if client == nil {
			return ne.DeviceNotConnected
		}

		// Set override on the client
		if duration > 0 {
			client.overrideUntil = time.Now().Add(duration)
			// Schedule clearing the override and sending channel's last image
			go func() {
				time.Sleep(duration)
				_ = RunTask(h.tasks, func() error {
					// Re-lookup client - it may have disconnected during sleep
					client, channel := h.findClientByDevice(deviceUUID)
					if client == nil {
						// Client disconnected, nothing to do
						return nil
					}
					// Check if override is still set and expired (not replaced by new push)
					if !client.overrideUntil.IsZero() && time.Now().After(client.overrideUntil) {
						client.overrideUntil = time.Time{}
						// Send the channel's last image immediately
						if channel != nil && channel.last != nil {
							client.send <- channel.last
						}
					}
					return nil
				})
			}()
		} else {
			// Indefinite override
			client.overrideUntil = time.Now().Add(24 * 365 * time.Hour)
		}

		// Send the image
		client.send <- &ClientImage{TTL: duration, Data: image}
		return nil
	})
}

// ClearDevicePush clears any active push override on a device and returns to channel content
func (h *Hub) ClearDevicePush(deviceUUID uuid.UUID) error {
	return RunTask(h.tasks, func() error {
		client, channel := h.findClientByDevice(deviceUUID)
		if client == nil {
			return ne.DeviceNotConnected
		}

		// Clear the override
		client.overrideUntil = time.Time{}

		// Send the channel's last image immediately
		if channel != nil && channel.last != nil {
			client.send <- channel.last
		}
		return nil
	})
}

// IsDeviceConnected checks if a device is currently connected
func (h *Hub) IsDeviceConnected(deviceUUID uuid.UUID) bool {
	connected := false
	_ = RunTask(h.tasks, func() error {
		client, _ := h.findClientByDevice(deviceUUID)
		connected = client != nil
		return nil
	})
	return connected
}
