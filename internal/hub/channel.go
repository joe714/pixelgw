package hub

import (
	"context"
	"crypto/md5"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/joe714/pixelgw/internal/catalog"
	"github.com/joe714/pixelgw/internal/locations"
	"tidbyt.dev/pixlet/encode"
	"tidbyt.dev/pixlet/runtime"
	"tidbyt.dev/pixlet/schema"
)

const (
	// TODO this should be a config option
	renderPeriod = 15 * time.Second
)

type AppConfig struct {
	UUID     uuid.UUID
	Manifest *catalog.Manifest
	Config   map[string]string `json:"config"`
	Ttl      time.Duration     `json:"ttl"`
}

type Channel struct {
	UUID          uuid.UUID
	Name          string
	hub           *Hub
	timer         *time.Timer
	tasks         chan *task
	clients       map[*Client]bool
	apps          []AppConfig
	nextApp       int
	last          *ClientImage
	overrideUntil time.Time // when set, pause render loop
}

func NewChannel(hub *Hub, uuid uuid.UUID, name string, apps []AppConfig) *Channel {
	ch := Channel{
		UUID:    uuid,
		Name:    name,
		hub:     hub,
		timer:   time.NewTimer(time.Nanosecond),
		tasks:   make(chan *task),
		clients: make(map[*Client]bool),
		apps:    apps,
		nextApp: 0,
		last:    nil,
	}
	return &ch
}

func (c *Channel) start() {
	go c.run()
}

func (c *Channel) run() {
	defer func() {
		c.timer.Stop()
	}()

	for {
		select {
		case <-c.timer.C:
			// Check if we're in override mode
			if !c.overrideUntil.IsZero() {
				remaining := time.Until(c.overrideUntil)
				if remaining > 0 {
					// Still in override, wait until it expires
					c.timer.Reset(remaining)
					continue
				}
				// Override expired, clear it
				c.overrideUntil = time.Time{}
			}

			buf, ttl := c.renderNext()
			if buf != nil {
				// TODO: redo the ttl / priority of channel images vs uploads
				c.last = &ClientImage{TTL: ttl, Data: buf}
				c.broadcastToClients(c.last)
			}
			c.timer.Reset(ttl)
		case task := <-c.tasks:
			task.run()
		}
	}
}

// broadcastToClients sends an image to all subscribed clients, respecting client overrides
func (c *Channel) broadcastToClients(img *ClientImage) {
	now := time.Now()
	for client := range c.clients {
		// Skip clients that have their own override active
		if !client.overrideUntil.IsZero() && client.overrideUntil.After(now) {
			continue
		}
		client.send <- img
	}
}

func (c *Channel) renderNext() ([]byte, time.Duration) {
	lim := len(c.apps)
	for i := 0; i < lim; i++ {
		app := c.apps[c.nextApp]
		c.nextApp = (c.nextApp + 1) % lim
		log.Printf("%v %v running\n", c.Name, app.Manifest.Name)
		applet, err := runtime.NewAppletFromFS(app.Manifest.ID, app.Manifest.Bundle)
		if err != nil {
			log.Printf("%v %v applet faild to load: %v\n", c.Name, app.Manifest.Name, err)
			continue
		}

		// Expand location configs to full location JSON
		config := expandLocationConfigs(applet.Schema, app.Config)

		roots, err := applet.RunWithConfig(context.Background(), config)
		if err != nil {
			log.Printf("%v %v applet failed: %v\n", c.Name, app.Manifest.Name, err)
			continue
		}
		if roots == nil || len(roots) < 1 {
			log.Printf("%v %v produced no roots\n", c.Name, app.Manifest.Name)
			continue
		}

		screens := encode.ScreensFromRoots(roots)
		img, err := screens.EncodeWebP(15000)
		if err != nil {
			log.Printf("%v %v encoding failed: %v\n", c.Name, app.Manifest.Name, err)
			continue
		}
		log.Printf("%v %v success (%v %x)\n", c.Name, app.Manifest.Name, len(img), md5.Sum(img))
		return img, renderPeriod // TODO make app.Ttl
	}
	log.Printf("%v ran out of render attempts\n", c.Name)
	return nil, renderPeriod
}

// expandLocationConfigs converts pixlet schema fields to location schema fields
// and delegates to the locations package for expansion
func expandLocationConfigs(sch *schema.Schema, config map[string]string) map[string]string {
	if sch == nil || len(config) == 0 {
		return config
	}

	// Convert pixlet schema fields to locations schema fields
	fields := make([]locations.SchemaField, 0, len(sch.Fields))
	for _, f := range sch.Fields {
		fields = append(fields, locations.SchemaField{
			ID:   f.ID,
			Type: f.Type,
		})
	}

	return locations.ExpandLocationConfigs(fields, config)
}

// pushContent sends an image to all clients and sets the channel override
func (c *Channel) pushContent(image []byte, duration time.Duration) error {
	return RunTask(c.tasks, func() error {
		// Set override (duration of 0 means indefinite until next render or push)
		if duration > 0 {
			c.overrideUntil = time.Now().Add(duration)
		} else {
			// Indefinite - set to a far future time
			c.overrideUntil = time.Now().Add(24 * 365 * time.Hour)
		}

		// Broadcast to all clients (including those with overrides since this is explicit)
		img := &ClientImage{TTL: duration, Data: image}
		for client := range c.clients {
			client.send <- img
		}

		// Reset timer to check override expiry
		if !c.timer.Stop() {
			select {
			case <-c.timer.C:
			default:
			}
		}
		if duration > 0 {
			c.timer.Reset(duration)
		}
		return nil
	})
}

// clearOverride clears any active push override and resumes normal render loop
func (c *Channel) clearOverride() error {
	return RunTask(c.tasks, func() error {
		c.overrideUntil = time.Time{}
		// Reset timer to fire immediately for next render
		if !c.timer.Stop() {
			select {
			case <-c.timer.C:
			default:
			}
		}
		c.timer.Reset(time.Nanosecond)
		return nil
	})
}

func (c *Channel) subscribe(client *Client) error {
	err := RunTask(c.tasks, func() error {
		c.clients[client] = true
		if c.last != nil {
			client.send <- c.last
		}
		return nil
	})
	return err
}

func (c *Channel) unsubscribe(client *Client) error {
	err := RunTask(c.tasks, func() error {
		delete(c.clients, client)
		return nil
	})
	return err
}

func (c *Channel) setApplets(apps []AppConfig, first uuid.UUID) error {
	idx := 0
	if first != uuid.Nil {
		for i, a := range apps {
			if a.UUID == first {
				idx = i
				break
			}
		}
	}

	err := RunTask(c.tasks, func() error {
		c.apps = apps
		c.nextApp = idx
		if !c.timer.Stop() {
			log.Printf("%v render now\n", c.Name)
			<-c.timer.C
		}
		c.timer.Reset(time.Nanosecond)
		return nil
	})
	return err
}

//func LoadClientConfig(catalog *catalog.Catalog, path string) ([]*AppConfig, error) {
//	cfgFile, err := os.Open("etc/clients/" + path)
//	if err != nil {
//		return nil, err
//	}
//
//	defer cfgFile.Close()
//	var tmp []*AppConfig
//	str, _ := ioutil.ReadAll(cfgFile)
//	json.Unmarshal([]byte(str), &tmp)
//	cfg := []*AppConfig{}
//	for _, e := range tmp {
//		e.manifest = catalog.FindManifest(e.AppId)
//		if e.manifest == nil {
//			log.Printf("%v: App \"%v\" not found in manifest\n", path, e.AppId)
//			continue
//		}
//		cfg = append(cfg, e)
//	}
//	return cfg, nil
//}
//
