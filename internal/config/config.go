package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// ConfigFileName is the name of the config file at project root
	ConfigFileName = ".beans.yml"
	// DefaultBeansPath is the default directory for storing beans
	DefaultBeansPath = ".beans"
	// LegacyConfigFile is the old config file location (deprecated)
	LegacyConfigFile = "config.yaml"
)

// DefaultStatuses defines the hardcoded status configuration.
// Statuses are not configurable - they are hardcoded like types.
// Order determines sort priority: in-progress first (active work), then todo, draft, and done states last.
var DefaultStatuses = []StatusConfig{
	{Name: "in-progress", Color: "yellow", Description: "Currently being worked on"},
	{Name: "todo", Color: "green", Description: "Ready to be worked on"},
	{Name: "draft", Color: "blue", Description: "Needs refinement before it can be worked on"},
	{Name: "completed", Color: "gray", Archive: true, Description: "Finished successfully"},
	{Name: "scrapped", Color: "gray", Archive: true, Description: "Will not be done"},
}

// DefaultTypes defines the default type configuration.
var DefaultTypes = []TypeConfig{
	{Name: "milestone", Color: "cyan", Description: "A target release or checkpoint; group work that should ship together"},
	{Name: "epic", Color: "purple", Description: "A thematic container for related work; should have child beans, not be worked on directly"},
	{Name: "bug", Color: "red", Description: "Something that is broken and needs fixing"},
	{Name: "feature", Color: "green", Description: "A user-facing capability or enhancement"},
	{Name: "task", Color: "blue", Description: "A concrete piece of work to complete (eg. a chore, or a sub-task for a feature)"},
}

// DefaultPriorities defines the hardcoded priority configuration.
// Priorities are ordered from highest to lowest urgency.
var DefaultPriorities = []PriorityConfig{
	{Name: "critical", Color: "red", Description: "Urgent, blocking work. When possible, address immediately"},
	{Name: "high", Color: "yellow", Description: "Important, should be done before normal work"},
	{Name: "normal", Color: "white", Description: "Standard priority"},
	{Name: "low", Color: "gray", Description: "Less important, can be delayed"},
	{Name: "deferred", Color: "gray", Description: "Explicitly pushed back, avoid doing unless necessary"},
}

// StatusConfig defines a single status with its display color.
type StatusConfig struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Archive     bool   `yaml:"archive,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// TypeConfig defines a single bean type with its display color.
type TypeConfig struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description,omitempty"`
}

// PriorityConfig defines a single priority level with its display color.
type PriorityConfig struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description,omitempty"`
}

// SessionConfig defines multiplexer-agnostic session management.
// Delegated to external tools (tmux, zellij, etc.) via scripts.
type SessionConfig struct {
	Check          string `yaml:"check"`            // Command that exits 0 if session exists
	Attach         string `yaml:"attach"`           // Command to attach to existing session
	Launch         string `yaml:"launch"`           // Command to launch/start a new session
	PollIntervalMs int    `yaml:"poll_interval_ms"` // Polling interval in ms (default: 200)
	CheckTimeoutMs int    `yaml:"check_timeout_ms"` // Timeout for check command in ms (default: 500)

	// Directories of the config files that declared each command (not serialized).
	// Commands run from there so relative script paths in inherited configs keep working.
	CheckDir  string `yaml:"-"`
	AttachDir string `yaml:"-"`
	LaunchDir string `yaml:"-"`
}

// Launcher defines a configured launcher command.
type Launcher struct {
	Name        string `yaml:"name"`
	Exec        string `yaml:"exec"`
	Description string `yaml:"description,omitempty"`
	Multiple    bool   `yaml:"multiple,omitempty"` // Supports launching for multiple beans in parallel

	// Dir is the directory of the config file that declared this launcher (not serialized).
	// The launcher runs from there so relative script paths in inherited configs keep working.
	Dir string `yaml:"-"`
}

// TUIConfig holds TUI-specific configuration.
type TUIConfig struct {
	// DisableLauncherWarning disables the confirmation prompt when launching for 5+ beans
	DisableLauncherWarning bool `yaml:"disable_launcher_warning,omitempty"`
}

// Config holds the beans configuration.
// Note: Statuses are no longer stored in config - they are hardcoded like types.
type Config struct {
	Beans     BeansConfig    `yaml:"beans"`
	TUI       TUIConfig      `yaml:"tui,omitempty"`
	Launchers []Launcher     `yaml:"launchers,omitempty"`
	Session   *SessionConfig `yaml:"session,omitempty"` // Multiplexer-agnostic session config

	// configDir is the directory containing the config file (not serialized)
	// Used to resolve relative paths
	configDir string `yaml:"-"`

	// configPath is the path to the config file it was loaded from (not serialized)
	configPath string `yaml:"-"`

	// beansPathDir is the directory of the config file that declared Beans.Path (not serialized).
	// Empty means Beans.Path resolves against configDir.
	beansPathDir string `yaml:"-"`
}

// BeansConfig defines settings for bean creation.
type BeansConfig struct {
	// Path is the path to the beans directory (relative to config file location, "~" expands to $HOME)
	Path          string `yaml:"path,omitempty"`
	Prefix        string `yaml:"prefix"`
	IDLength      int    `yaml:"id_length"`
	DefaultStatus string `yaml:"default_status,omitempty"`
	DefaultType   string `yaml:"default_type,omitempty"`
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		Beans: BeansConfig{
			Path:          DefaultBeansPath,
			Prefix:        "",
			IDLength:      4,
			DefaultStatus: "todo",
			DefaultType:   "task",
		},
	}
}

// DefaultWithPrefix returns a Config with the given prefix.
func DefaultWithPrefix(prefix string) *Config {
	cfg := Default()
	cfg.Beans.Prefix = prefix
	return cfg
}

// FindConfig searches upward from the given directory for a .beans.yml config file.
// Returns the absolute path to the config file, or empty string if not found.
func FindConfig(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		configPath := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", nil
		}
		dir = parent
	}
}

// Load reads configuration from the given config file path, merged with the
// settings of cascading ancestor configs (see findChain).
// Returns default config if the file doesn't exist.
func Load(configPath string) (*Config, error) {
	layers, err := findChain(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}

	return fromLayers(configPath, layers)
}

// fromLayers builds the config for configPath from its layers (root-first).
func fromLayers(configPath string, layers []layer) (*Config, error) {
	cfg, err := decode(mergeLayers(layers))
	if err != nil {
		return nil, err
	}

	cfg.configDir = filepath.Dir(configPath)
	cfg.configPath = configPath
	cfg.recordOrigins(layers)
	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// mergeLayers merges all layers root-first. The cascade setting is per-file
// and is dropped from the result.
func mergeLayers(layers []layer) map[string]any {
	merged := map[string]any{}
	for _, l := range layers {
		merged = mergeData(merged, l.data)
	}
	delete(merged, cascadeKey)
	return merged
}

// decode turns merged settings into a Config.
func decode(data map[string]any) (*Config, error) {
	raw, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// recordOrigins remembers which config file declared path-dependent settings,
// so they resolve against the file that wrote them.
func (c *Config) recordOrigins(layers []layer) {
	c.beansPathDir = originDir(layers, "beans", "path")

	for i := range c.Launchers {
		dir := originDirOfNamed(layers, "launchers", c.Launchers[i].Name)
		if dir == "" {
			dir = originDir(layers, "launchers")
		}
		c.Launchers[i].Dir = dir
	}

	if c.Session != nil {
		c.Session.CheckDir = originDir(layers, "session", "check")
		c.Session.AttachDir = originDir(layers, "session", "attach")
		c.Session.LaunchDir = originDir(layers, "session", "launch")
	}
}

// applyDefaults fills in missing values.
func (c *Config) applyDefaults() {
	if c.Beans.Path == "" {
		c.Beans.Path = DefaultBeansPath
	}
	if c.Beans.IDLength == 0 {
		c.Beans.IDLength = 4
	}
	if c.Beans.DefaultStatus == "" {
		c.Beans.DefaultStatus = "todo"
	}
	if c.Beans.DefaultType == "" {
		c.Beans.DefaultType = DefaultTypes[0].Name
	}
}

// validate checks the merged config for errors.
func (c *Config) validate() error {
	for _, launcher := range c.Launchers {
		// Multi-line exec must have shebang
		if strings.Contains(launcher.Exec, "\n") && !strings.HasPrefix(launcher.Exec, "#!") {
			return fmt.Errorf("launcher '%s' has multi-line exec but no shebang (#!). Multi-line scripts must start with a shebang (e.g., #!/bin/bash)", launcher.Name)
		}
	}
	return nil
}

// LoadFromDirectory finds and loads the config file by searching upward from the given directory.
// If no config file is found, returns a default config anchored at the given directory.
func LoadFromDirectory(startDir string) (*Config, error) {
	configPath, err := FindConfig(startDir)
	if err != nil {
		return nil, err
	}

	if configPath == "" {
		// No config found, return default anchored at startDir
		cfg := Default()
		cfg.configDir = startDir
		return cfg, nil
	}

	return Load(configPath)
}

// ExpandHome replaces a leading "~" in path with the user's home directory.
// Only "~" and "~/..." are expanded; "~user" forms are returned unchanged.
func ExpandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

// ResolveBeansPath returns the absolute path to the beans directory.
func (c *Config) ResolveBeansPath() string {
	path := ExpandHome(c.Beans.Path)
	if filepath.IsAbs(path) {
		return path
	}
	if c.beansPathDir != "" {
		return filepath.Join(c.beansPathDir, path)
	}
	if c.configDir == "" {
		// Fallback: use current directory
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, path)
	}
	return filepath.Join(c.configDir, path)
}

// ConfigDir returns the directory containing the config file.
func (c *Config) ConfigDir() string {
	return c.configDir
}

// ConfigPath returns the path to the config file.
// Falls back to ConfigFileName inside the config directory if the config
// was not loaded from a file.
func (c *Config) ConfigPath() string {
	if c.configPath != "" {
		return c.configPath
	}
	return filepath.Join(c.configDir, ConfigFileName)
}

// SetConfigDir sets the config directory (for testing or when creating new configs).
func (c *Config) SetConfigDir(dir string) {
	c.configDir = dir
}

// Save writes the configuration to the config file.
// It must not be called on a config loaded with cascading ancestors: it would
// write all inherited settings into the leaf file.
// If configDir is set, saves to that directory; otherwise saves to the given directory.
func (c *Config) Save(dir string) error {
	targetDir := c.configDir
	if targetDir == "" {
		targetDir = dir
	}
	path := filepath.Join(targetDir, ConfigFileName)

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// IsValidStatus returns true if the status is a valid hardcoded status.
func (c *Config) IsValidStatus(status string) bool {
	for _, s := range DefaultStatuses {
		if s.Name == status {
			return true
		}
	}
	return false
}

// StatusList returns a comma-separated list of valid statuses.
// Statuses are hardcoded and not configurable.
func (c *Config) StatusList() string {
	names := make([]string, len(DefaultStatuses))
	for i, s := range DefaultStatuses {
		names[i] = s.Name
	}
	return strings.Join(names, ", ")
}

// StatusNames returns a slice of valid status names.
// Statuses are hardcoded and not configurable.
func (c *Config) StatusNames() []string {
	names := make([]string, len(DefaultStatuses))
	for i, s := range DefaultStatuses {
		names[i] = s.Name
	}
	return names
}

// GetStatus returns the StatusConfig for a given status name, or nil if not found.
// Statuses are hardcoded and not configurable.
func (c *Config) GetStatus(name string) *StatusConfig {
	for i := range DefaultStatuses {
		if DefaultStatuses[i].Name == name {
			return &DefaultStatuses[i]
		}
	}
	return nil
}

// GetDefaultStatus returns the default status name for new beans.
func (c *Config) GetDefaultStatus() string {
	if c.Beans.DefaultStatus == "" {
		return "todo"
	}
	return c.Beans.DefaultStatus
}

// GetDefaultType returns the default type name for new beans.
func (c *Config) GetDefaultType() string {
	return c.Beans.DefaultType
}

// IsArchiveStatus returns true if the given status is marked for archiving.
// Statuses are hardcoded and not configurable.
func (c *Config) IsArchiveStatus(name string) bool {
	if s := c.GetStatus(name); s != nil {
		return s.Archive
	}
	return false
}

// GetType returns the TypeConfig for a given type name, or nil if not found.
// Types are hardcoded and not configurable.
func (c *Config) GetType(name string) *TypeConfig {
	for i := range DefaultTypes {
		if DefaultTypes[i].Name == name {
			return &DefaultTypes[i]
		}
	}
	return nil
}

// TypeNames returns a slice of valid type names.
// Types are hardcoded and not configurable.
func (c *Config) TypeNames() []string {
	names := make([]string, len(DefaultTypes))
	for i, t := range DefaultTypes {
		names[i] = t.Name
	}
	return names
}

// IsValidType returns true if the type is a valid hardcoded type.
func (c *Config) IsValidType(typeName string) bool {
	for _, t := range DefaultTypes {
		if t.Name == typeName {
			return true
		}
	}
	return false
}

// TypeList returns a comma-separated list of valid types.
func (c *Config) TypeList() string {
	names := make([]string, len(DefaultTypes))
	for i, t := range DefaultTypes {
		names[i] = t.Name
	}
	return strings.Join(names, ", ")
}

// BeanColors holds resolved color information for rendering a bean
type BeanColors struct {
	StatusColor   string
	TypeColor     string
	PriorityColor string
	IsArchive     bool
}

// GetBeanColors returns the resolved colors for a bean based on its status, type, and priority.
func (c *Config) GetBeanColors(status, typeName, priority string) BeanColors {
	colors := BeanColors{
		StatusColor:   "gray",
		TypeColor:     "",
		PriorityColor: "",
		IsArchive:     false,
	}

	if statusCfg := c.GetStatus(status); statusCfg != nil {
		colors.StatusColor = statusCfg.Color
	}
	colors.IsArchive = c.IsArchiveStatus(status)

	if typeCfg := c.GetType(typeName); typeCfg != nil {
		colors.TypeColor = typeCfg.Color
	}

	if priorityCfg := c.GetPriority(priority); priorityCfg != nil {
		colors.PriorityColor = priorityCfg.Color
	}

	return colors
}

// GetPriority returns the PriorityConfig for a given priority name, or nil if not found.
func (c *Config) GetPriority(name string) *PriorityConfig {
	for i := range DefaultPriorities {
		if DefaultPriorities[i].Name == name {
			return &DefaultPriorities[i]
		}
	}
	return nil
}

// PriorityNames returns a slice of valid priority names in order from highest to lowest.
func (c *Config) PriorityNames() []string {
	names := make([]string, len(DefaultPriorities))
	for i, p := range DefaultPriorities {
		names[i] = p.Name
	}
	return names
}

// IsValidPriority returns true if the priority is a valid hardcoded priority.
// Empty string is valid (means no priority set).
func (c *Config) IsValidPriority(priority string) bool {
	if priority == "" {
		return true
	}
	for _, p := range DefaultPriorities {
		if p.Name == priority {
			return true
		}
	}
	return false
}

// PriorityList returns a comma-separated list of valid priorities.
func (c *Config) PriorityList() string {
	names := make([]string, len(DefaultPriorities))
	for i, p := range DefaultPriorities {
		names[i] = p.Name
	}
	return strings.Join(names, ", ")
}
