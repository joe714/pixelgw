package hub

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/google/uuid"

	"github.com/joe714/pixelgw/internal/catalog"
	"github.com/joe714/pixelgw/internal/durable"
)

type SessionInfo struct {
	SessionID   uint32
	ClientUUID  uuid.UUID
	RemoteAddr  string
	ChannelUUID uuid.UUID
	ChannelName string
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
					app.Config,
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
//	clientId := r.FormValue("clientId")
//	data, err := ioutil.ReadAll(file)
//	log.Printf("%v: upload for %v %v", host, clientId, len(data))
//	if err != nil {
//		log.Printf("%v upload read failed: %v", host, err)
//		return
//	}
//
//	msg := &BroadcastMsg{clientId: clientId, ttl: 15 * time.Second, data: data}
//	h.broadcast <- msg
//}

func (h *Hub) wsHandler(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	q := r.URL.Query()
	id := q.Get("device")
	if len(id) == 0 {
		// Fallback for older firmare
		id = q.Get("clientId")
	}

	if len(id) == 0 {
		log.Printf("%v: No device UUID specified", host)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	deviceUUID, err := uuid.Parse(id)
	if err != nil {
		log.Println("%v %v: Device UUID is not valid: %v", id, host, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	device, err := h.store.LoginDevice(r.Context(), deviceUUID)
	if err != nil {
		log.Println("%v %v: failed to get device configuration: %v", deviceUUID, host, err)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("%v %v: failed to establish websocket: %v", deviceUUID, host, err)
		return
	}
	client := NewClient(deviceUUID, conn)
	log.Printf("%v established from %v", client, host)
	_ = h.register(client, device.ChannelUUID)
}

func (h *Hub) GetWsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { h.wsHandler(w, r) }
}

func (h *Hub) GetSessions() []SessionInfo {
	resp := []SessionInfo{}
	_ = RunTask(h.tasks, func() error {
		for k, v := range h.clients {
			addr, _, _ := net.SplitHostPort(k.RemoteAddr().String())
			resp = append(resp, SessionInfo{
				SessionID:   k.SessionID,
				ClientUUID:  k.UUID,
				RemoteAddr:  addr,
				ChannelUUID: v.UUID,
				ChannelName: v.Name,
			})
		}
		return nil
	})
	return resp
}
