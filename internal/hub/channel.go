package hub

import (
	"context"
	"crypto/md5"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/joe714/pixelgw/internal/catalog"
	"tidbyt.dev/pixlet/encode"
	"tidbyt.dev/pixlet/runtime"
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
	UUID    uuid.UUID
	Name    string
	hub     *Hub
	timer   *time.Timer
	tasks   chan *task
	clients map[*Client]bool
	apps    []AppConfig
	nextApp int
	last    *ClientImage
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
			buf, ttl := c.renderNext()
			if buf != nil {
				// TODO: redo the ttl / priority of channel images vs uploads
				c.last = &ClientImage{ttl: ttl, data: buf}
				for client, _ := range c.clients {
					client.send <- c.last
				}
			}
			c.timer.Reset(ttl)
		case task := <-c.tasks:
			task.run()
		}
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
		roots, err := applet.RunWithConfig(context.Background(), app.Config)
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
			log.Printf("%v %v encoding failed %v: %v\n", c.Name, app.Manifest.Name, err)
			continue
		}
		log.Printf("%v %v success (%v %x)\n", c.Name, app.Manifest.Name, len(img), md5.Sum(img))
		return img, renderPeriod // TODO make app.Ttl
	}
	log.Printf("%v ran out of render attempts\n", c.Name)
	return nil, renderPeriod
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
