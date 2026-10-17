package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// UserConfigFileName is the name of the user-global config file inside the
// user config directory.
const UserConfigFileName = "config.yml"

// UserConfigPath returns the path of the user-global config file:
// $XDG_CONFIG_HOME/beans/config.yml, falling back to ~/.config/beans/config.yml.
// Returns "" if neither location can be determined.
func UserConfigPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "beans", UserConfigFileName)
}

// readUserLayer reads the user-global config file. It is the base layer below
// every project config. ok is false if the file doesn't exist.
func readUserLayer() (l layer, ok bool, err error) {
	path := UserConfigPath()
	if path == "" {
		return layer{}, false, nil
	}

	l, err = readLayer(path)
	if os.IsNotExist(err) {
		return layer{}, false, nil
	}
	if err != nil {
		return layer{}, false, fmt.Errorf("reading %s: %w", path, err)
	}

	// The user config always applies; cascading is meaningless here
	delete(l.data, cascadeKey)

	if hasKeyPath(l.data, []string{"beans", "path"}) {
		return layer{}, false, fmt.Errorf("%s: beans.path can't be set in the user config", path)
	}
	return l, true, nil
}

// withUserLayer prepends the user-global layer (if any) to layers.
func withUserLayer(layers []layer) ([]layer, error) {
	user, ok, err := readUserLayer()
	if err != nil || !ok {
		return layers, err
	}
	return append([]layer{user}, layers...), nil
}
