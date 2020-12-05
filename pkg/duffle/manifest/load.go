package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	// DuffleFilename is the default filename for Duffle application configuration.
	DuffleFilename = "duffle"
)

// Load parses the named file into a manifest.
func Load(name, dir string) (*Manifest, error) {
	if name == "" {
		name = DuffleFilename + ".json"
	}
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return nil, err
	}

	manifest := New()
	if err := json.NewDecoder(f).Decode(&manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}
