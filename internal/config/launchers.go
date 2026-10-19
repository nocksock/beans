package config

import (
	"fmt"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"
)

// launcherLayer is the launchers one config file declares, before merging.
type launcherLayer struct {
	dir       string
	launchers []Launcher
}

// decodeLauncherLayers reads the launchers each layer declares (root-first),
// each tagged with its layer's directory.
func decodeLauncherLayers(layers []layer) ([]launcherLayer, error) {
	result := make([]launcherLayer, 0, len(layers))
	for _, l := range layers {
		ll := launcherLayer{dir: l.dir}
		if items, ok := l.data["launchers"]; ok {
			raw, err := yaml.Marshal(items)
			if err != nil {
				return nil, err
			}
			if err := yaml.Unmarshal(raw, &ll.launchers); err != nil {
				return nil, fmt.Errorf("launchers in %s: %w", l.dir, err)
			}
			for i := range ll.launchers {
				ll.launchers[i].Dir = l.dir
			}
		}
		result = append(result, ll)
	}
	return result, nil
}

// Launcher returns the launcher with the given name from the merged config.
func (c *Config) Launcher(name string) (Launcher, bool) {
	i := slices.IndexFunc(c.Launchers, func(l Launcher) bool { return l.Name == name })
	if i < 0 {
		return Launcher{}, false
	}
	return c.Launchers[i], true
}

// ParentLauncher returns the launcher name as declared by the nearest config
// layer above the caller's: the definition a launcher in that layer
// overrides. ok is false when no layer above declares it.
//
// layerDir names the caller's config layer (compared after filepath.Clean, as
// it usually comes back from an environment variable). An empty layerDir means
// the caller is the leaf: with a project config file the leaf is the last
// layer; without one (only the user layer, if any) every layer is above it.
func (c *Config) ParentLauncher(name, layerDir string) (l Launcher, ok bool, err error) {
	var start int
	if layerDir == "" {
		start = len(c.launcherLayers) - 1
		if c.configPath != "" {
			start--
		}
	} else {
		want := filepath.Clean(layerDir)
		i := slices.IndexFunc(c.launcherLayers, func(ll launcherLayer) bool { return filepath.Clean(ll.dir) == want })
		if i < 0 {
			return Launcher{}, false, fmt.Errorf("config directory %s is not part of this config", layerDir)
		}
		start = i - 1
	}
	for i := start; i >= 0; i-- {
		for _, l := range c.launcherLayers[i].launchers {
			if l.Name == name {
				return l, true, nil
			}
		}
	}
	return Launcher{}, false, nil
}

// LoadedPath returns the config file this config was loaded from, or "" when
// no project config file was found. Unlike ConfigPath it has no fallback.
func (c *Config) LoadedPath() string {
	return c.configPath
}
