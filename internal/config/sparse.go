package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// CascadingParent returns the path of the nearest config file above dir if
// that config cascades its settings down into dir, or "" if there is none.
func CascadingParent(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	parentPath, err := findConfigAbove(abs)
	if err != nil || parentPath == "" {
		return "", err
	}
	parent, err := readLayer(parentPath)
	if err != nil {
		return "", err
	}
	if !parent.cascades() {
		return "", nil
	}
	return parentPath, nil
}

// SaveSettings writes a config file to dir that holds only the given
// settings, keyed by dotted path like "beans.prefix". Everything else is
// left to be inherited from a cascading parent.
func SaveSettings(dir string, settings map[string]any) error {
	data := map[string]any{}
	for key, value := range settings {
		nestSetting(data, strings.Split(key, "."), value)
	}

	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(data); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ConfigFileName), out.Bytes(), 0644)
}

// nestSetting stores value in data under the nested keys, creating
// intermediate maps as needed.
func nestSetting(data map[string]any, keys []string, value any) {
	if len(keys) == 1 {
		data[keys[0]] = value
		return
	}
	child, ok := data[keys[0]].(map[string]any)
	if !ok {
		child = map[string]any{}
		data[keys[0]] = child
	}
	nestSetting(child, keys[1:], value)
}
