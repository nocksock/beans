package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// cascadeKey is the per-file setting that passes a file's settings down to
// configs in its subdirectories. It is never inherited.
const cascadeKey = "cascade"

// layer is one config file's raw settings, before merging.
type layer struct {
	dir  string         // directory containing the file
	data map[string]any // parsed settings
}

// readLayer reads one config file into a layer.
// This is the only place that knows the file format.
func readLayer(path string) (layer, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return layer{}, err
	}

	var data map[string]any
	if err := yaml.Unmarshal(raw, &data); err != nil {
		return layer{}, err
	}

	return layer{dir: filepath.Dir(path), data: data}, nil
}

// cascades reports whether this layer passes its settings down.
func (l layer) cascades() bool {
	v, _ := l.data[cascadeKey].(bool)
	return v
}
