package pixelclient

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Device represents a configured device entry.
type Device struct {
	Name   string `json:"name"`
	ID     string `json:"id"`
	Server string `json:"server"`
}

// Config holds the list of configured devices and the path it was loaded from.
type Config struct {
	Devices []Device `json:"devices"`
	path    string   // not serialized, tracks the file path
}

// DefaultConfigPath returns the default path to the config file.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pixelclient", "config.json"), nil
}

// LoadConfig reads the config file from the specified path.
// If path is empty, uses the default path (~/.pixelclient/config.json).
// Returns an empty config if the file doesn't exist.
func LoadConfig(path string) (*Config, error) {
	var err error
	if path == "" {
		path, err = DefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	cfg := &Config{
		Devices: []Device{},
		path:    path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the config back to the path it was loaded from.
func (c *Config) Save() error {
	if c.path == "" {
		var err error
		c.path, err = DefaultConfigPath()
		if err != nil {
			return err
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0644)
}

// Path returns the config file path.
func (c *Config) Path() string {
	return c.path
}

// FindDeviceByName returns the device with the given name, or nil if not found.
func (c *Config) FindDeviceByName(name string) *Device {
	for i := range c.Devices {
		if c.Devices[i].Name == name {
			return &c.Devices[i]
		}
	}
	return nil
}

// FindDeviceByID returns the device with the given ID, or nil if not found.
func (c *Config) FindDeviceByID(id string) *Device {
	for i := range c.Devices {
		if c.Devices[i].ID == id {
			return &c.Devices[i]
		}
	}
	return nil
}

// AddDevice adds a new device to the config.
// Returns an error if name or ID already exists, or if server is invalid.
func (c *Config) AddDevice(name, server, deviceID string) (*Device, error) {
	if c.FindDeviceByName(name) != nil {
		return nil, errors.New("device name already exists")
	}

	// Validate server format (must be host:port, no scheme)
	if strings.Contains(server, "://") {
		return nil, errors.New("server must be host:port (no http:// or ws:// prefix)")
	}
	host, port, err := net.SplitHostPort(server)
	if err != nil {
		return nil, errors.New("server must be host:port format (e.g., localhost:8080)")
	}
	if host == "" || port == "" {
		return nil, errors.New("server must include both host and port")
	}

	if deviceID == "" {
		deviceID = uuid.New().String()
	} else {
		// Validate UUID format
		if _, err := uuid.Parse(deviceID); err != nil {
			return nil, errors.New("invalid device ID format (must be UUID)")
		}
		if c.FindDeviceByID(deviceID) != nil {
			return nil, errors.New("device ID already exists")
		}
	}

	device := Device{
		Name:   name,
		ID:     deviceID,
		Server: server,
	}
	c.Devices = append(c.Devices, device)
	return &device, nil
}

// GetDefaultDevice returns the device named "default", or nil if not found.
func (c *Config) GetDefaultDevice() *Device {
	return c.FindDeviceByName("default")
}
