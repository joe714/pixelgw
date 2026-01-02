package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"

	"tidbyt.dev/pixlet/encode"
	"tidbyt.dev/pixlet/manifest"
	"tidbyt.dev/pixlet/runtime"
)

type Manifest struct {
	manifest.Manifest
	Bundle fs.FS
}

type Catalog struct {
	Manifests map[string]*Manifest
}

func NewCatalog(root fs.FS) *Catalog {
	matches, err := fs.Glob(root, "*/manifest.yaml")
	if err != nil {
		log.Printf("Failed to find manifest files: %v\n", err)
		return nil
	}

	catalog := &Catalog{make(map[string]*Manifest)}

	for _, m := range matches {
		in, err := root.Open(m)
		if err != nil {
			log.Printf("Failed to open manifest %v: %v\n", m, err)
			continue
		}
		mn, err := manifest.LoadManifest(in)
		if err != nil {
			log.Printf("Failed to load manifest %v: %v\n", m, err)
			continue
		}
		bundle, err := fs.Sub(root, filepath.Dir(m))
		if err != nil {
			log.Printf("Failed to create bundle handle %v: %v\n", m, err)
			continue
		}

		catalog.Manifests[mn.ID] = &Manifest{
			Manifest: *mn,
			Bundle:   bundle,
		}
		log.Printf("Loaded app %v from %v", mn.ID, m)
	}
	return catalog
}

func (c *Catalog) FindManifest(id string) *Manifest {
	m, _ := c.Manifests[id]
	return m
}

// RenderApplet renders an applet with the given config and returns WebP image data
func (c *Catalog) RenderApplet(m *Manifest, config map[string]interface{}) ([]byte, error) {
	applet, err := runtime.NewAppletFromFS(m.ID, m.Bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to load applet: %w", err)
	}

	// Convert config to string map for pixlet
	stringConfig := make(map[string]string)
	if config != nil {
		for k, v := range config {
			switch val := v.(type) {
			case string:
				stringConfig[k] = val
			case float64:
				stringConfig[k] = fmt.Sprintf("%v", val)
			case bool:
				stringConfig[k] = fmt.Sprintf("%v", val)
			case map[string]interface{}:
				// For location objects, serialize to JSON
				jsonBytes, err := json.Marshal(val)
				if err != nil {
					return nil, fmt.Errorf("failed to serialize config value for %s: %w", k, err)
				}
				stringConfig[k] = string(jsonBytes)
			default:
				stringConfig[k] = fmt.Sprintf("%v", val)
			}
		}
	}

	roots, err := applet.RunWithConfig(context.Background(), stringConfig)
	if err != nil {
		return nil, fmt.Errorf("applet execution failed: %w", err)
	}
	if roots == nil || len(roots) < 1 {
		return nil, fmt.Errorf("applet produced no output")
	}

	screens := encode.ScreensFromRoots(roots)
	img, err := screens.EncodeWebP(15000)
	if err != nil {
		return nil, fmt.Errorf("failed to encode WebP: %w", err)
	}

	return img, nil
}
