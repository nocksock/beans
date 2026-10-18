package config

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Scope returns the bean ID prefix that lists are limited to, or "" for no
// limit. A config inheriting from a cascading ancestor is scoped to its own
// prefix; the root of a cascade (or a standalone config) sees everything.
func (c *Config) Scope() string {
	if !c.inherited {
		return ""
	}
	return c.Beans.Prefix
}

// AvailablePrefixes returns the prefixes new beans can be created with: this
// config's own prefix first, followed by those of descendant configs that
// cascade from it and share its beans directory.
func (c *Config) AvailablePrefixes() []string {
	prefixes := []string{c.Beans.Prefix}
	for _, p := range c.descendantConfigs() {
		sub, err := Load(p)
		if err != nil || sub.ResolveBeansPath() != c.ResolveBeansPath() {
			continue
		}
		if sub.Beans.Prefix != "" && !slices.Contains(prefixes, sub.Beans.Prefix) {
			prefixes = append(prefixes, sub.Beans.Prefix)
		}
	}
	return prefixes
}

// descendantConfigs returns the config files below this config's directory
// that inherit from it, i.e. every config between them cascades.
func (c *Config) descendantConfigs() []string {
	if c.configPath == "" {
		return nil
	}
	own, err := readLayer(c.configPath)
	if err != nil || !own.cascades() {
		return nil
	}

	beansDir := c.ResolveBeansPath()
	var found []string
	_ = filepath.WalkDir(c.configDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if path == c.configDir {
			return nil
		}
		if skipDir(d.Name()) || path == beansDir {
			return filepath.SkipDir
		}

		configPath := filepath.Join(path, ConfigFileName)
		if _, err := os.Stat(configPath); err != nil {
			return nil
		}
		found = append(found, configPath)

		// Configs below a non-cascading one don't inherit from us
		l, err := readLayer(configPath)
		if err != nil || !l.cascades() {
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

// skipDir reports whether a directory is never searched for configs.
func skipDir(name string) bool {
	return strings.HasPrefix(name, ".") || name == "node_modules"
}
