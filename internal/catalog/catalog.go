package catalog

import (
	"io/fs"
	"log"
	"path/filepath"

	"tidbyt.dev/pixlet/manifest"
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
