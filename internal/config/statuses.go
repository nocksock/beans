package config

import (
	"fmt"
	"slices"
	"sort"

	"gopkg.in/yaml.v3"
)

// CustomStatuses maps a builtin status to the custom statuses grouped below it.
type CustomStatuses map[string][]CustomStatusConfig

// UnmarshalYAML ignores the legacy list form of `statuses` (from when statuses
// were configurable without grouping), so old config files keep loading.
func (cs *CustomStatuses) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.SequenceNode {
		return nil
	}
	var m map[string][]CustomStatusConfig
	if err := node.Decode(&m); err != nil {
		return err
	}
	*cs = m
	return nil
}

// Statuses returns all statuses in sort order: each builtin status followed by
// the custom statuses grouped below it.
func (c *Config) Statuses() []StatusConfig {
	if c == nil || c.statuses == nil {
		return DefaultStatuses
	}
	return c.statuses
}

// StatusCategory returns the builtin status a status belongs to: its parent for
// a custom status, itself for a builtin, and "" for an unknown status.
func (c *Config) StatusCategory(name string) string {
	s := c.GetStatus(name)
	if s == nil {
		return ""
	}
	if s.Parent != "" {
		return s.Parent
	}
	return s.Name
}

// ExpandStatuses adds the custom statuses below each builtin status in names,
// so that filtering by a builtin status also matches its substatuses.
func (c *Config) ExpandStatuses(names []string) []string {
	if len(names) == 0 {
		return names
	}
	result := make([]string, 0, len(names))
	add := func(name string) {
		if !slices.Contains(result, name) {
			result = append(result, name)
		}
	}
	for _, name := range names {
		add(name)
		for _, s := range c.Statuses() {
			if s.Parent != "" && s.Parent == name {
				add(s.Name)
			}
		}
	}
	return result
}

// resolveStatuses validates the custom statuses and returns the full status
// list, or nil when there are no custom statuses.
func resolveStatuses(custom CustomStatuses) ([]StatusConfig, error) {
	if len(custom) == 0 {
		return nil, nil
	}

	builtins := make(map[string]bool, len(DefaultStatuses))
	for _, s := range DefaultStatuses {
		builtins[s.Name] = true
	}

	// Check in a stable order so errors are deterministic
	parents := make([]string, 0, len(custom))
	for parent := range custom {
		parents = append(parents, parent)
	}
	sort.Strings(parents)

	seen := map[string]bool{}
	for _, parent := range parents {
		if !builtins[parent] {
			return nil, fmt.Errorf("statuses: '%s' is not a builtin status (custom statuses must be grouped below one of: %s)", parent, builtinStatusList())
		}
		for _, s := range custom[parent] {
			switch {
			case s.Name == "":
				return nil, fmt.Errorf("statuses: every custom status below '%s' needs a name", parent)
			case builtins[s.Name]:
				return nil, fmt.Errorf("statuses: '%s' is a builtin status and can't be redefined", s.Name)
			case seen[s.Name]:
				return nil, fmt.Errorf("statuses: '%s' is defined more than once", s.Name)
			}
			seen[s.Name] = true
		}
	}

	result := make([]StatusConfig, 0, len(DefaultStatuses)+len(seen))
	for _, builtin := range DefaultStatuses {
		result = append(result, builtin)
		for _, s := range custom[builtin.Name] {
			color := s.Color
			if color == "" {
				color = builtin.Color
			}
			result = append(result, StatusConfig{
				Name:        s.Name,
				Color:       color,
				Archive:     builtin.Archive,
				Description: s.Description,
				Parent:      builtin.Name,
			})
		}
	}
	return result, nil
}

func builtinStatusList() string {
	var cfg *Config
	return cfg.StatusList()
}
