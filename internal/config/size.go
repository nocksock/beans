package config

import "strings"

// DefaultSize is the size a bean without an explicit size counts as.
const DefaultSize = "md"

// DefaultSizes defines the hardcoded size configuration.
// Sizes are ordered from smallest to largest.
var DefaultSizes = []SizeConfig{
	{Name: "xs", Color: "cyan", Description: "Trivial, a few minutes of work"},
	{Name: "sm", Color: "green", Description: "Small, well-understood change"},
	{Name: "md", Color: "white", Description: "Standard size"},
	{Name: "lg", Color: "yellow", Description: "Large, touches several areas"},
	{Name: "xl", Color: "red", Description: "Very large, consider splitting it up"},
}

// SizeConfig defines a single size with its display color.
type SizeConfig struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description,omitempty"`
}

// EffectiveSize returns size, or DefaultSize when size is empty.
func EffectiveSize(size string) string {
	if size == "" {
		return DefaultSize
	}
	return size
}

// GetSize returns the SizeConfig for a given size name, or nil if not found.
// An empty name resolves to DefaultSize.
func (c *Config) GetSize(name string) *SizeConfig {
	name = EffectiveSize(name)
	for i := range DefaultSizes {
		if DefaultSizes[i].Name == name {
			return &DefaultSizes[i]
		}
	}
	return nil
}

// SizeNames returns a slice of valid size names in order from smallest to largest.
func (c *Config) SizeNames() []string {
	names := make([]string, len(DefaultSizes))
	for i, s := range DefaultSizes {
		names[i] = s.Name
	}
	return names
}

// IsValidSize returns true if the size is a valid hardcoded size.
// Empty string is valid (means no size set, i.e. DefaultSize).
func (c *Config) IsValidSize(size string) bool {
	if size == "" {
		return true
	}
	return c.GetSize(size) != nil
}

// SizeList returns a comma-separated list of valid sizes.
func (c *Config) SizeList() string {
	return strings.Join(c.SizeNames(), ", ")
}
