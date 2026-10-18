package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	initJSON          bool
	initInherit       bool
	initCascade       bool
	initForce         bool
	initPath          string
	initPrefix        string
	initIDLength      int
	initDefaultStatus string
	initDefaultType   string
	initPromptExtra   string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a beans project",
	Long: `Creates a .beans directory and .beans.yml config file in the current directory
(or next to --beans-path). An existing .beans.yml is only overwritten with --force.

Flags set the matching .beans.yml settings. For example, to keep the beans
outside the project:

  beans init --path ~/path/to/tasks

Pass --cascade to let configs in subdirectories inherit this one (sets cascade: true).

Inside a directory that a parent config cascades into (cascade: true), init
refuses to run unless --inherit is given. With --inherit it writes a config
that holds only the settings passed as flags; everything else keeps coming
from the parent:

  beans init --inherit --prefix api-`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if beansPath != "" && cmd.Flags().Changed("path") {
			return initFail(output.ErrValidation, "invalid flags", fmt.Errorf("--path and --beans-path cannot be combined"))
		}

		var projectDir string
		if beansPath != "" {
			// The beans directory is given explicitly; the config goes next to it
			projectDir = filepath.Dir(beansPath)
		} else {
			dir, err := os.Getwd()
			if err != nil {
				return initFail(output.ErrFileError, "failed to get current directory", err)
			}
			projectDir = dir
		}

		configFile := filepath.Join(projectDir, config.ConfigFileName)
		if _, err := os.Stat(configFile); err == nil && !initForce {
			return initFail(output.ErrValidation, "refusing to init", fmt.Errorf("%s already exists; pass --force to overwrite it", configFile))
		}

		parentPath, err := config.CascadingParent(projectDir)
		if err != nil {
			return initFail(output.ErrFileError, "failed to read parent config", err)
		}
		if parentPath != "" && !initInherit {
			return initFail(output.ErrValidation, "refusing to init", fmt.Errorf(
				"%s cascades into this directory; pass --inherit to create a config that only sets the given flags", parentPath))
		}
		if initInherit && parentPath == "" {
			return initFail(output.ErrValidation, "invalid flags", fmt.Errorf("--inherit requires a parent config with cascade: true"))
		}

		var beansDir string
		if initInherit {
			beansDir, err = initInherited(projectDir, parentPath, cmd.Flags())
		} else {
			beansDir, err = initStandalone(projectDir, cmd.Flags())
		}
		if err != nil {
			return err
		}

		if initJSON {
			return output.SuccessInit(beansDir)
		}

		fmt.Println("Initialized beans project")
		return nil
	},
}

// initFail wraps an init error for the chosen output mode.
func initFail(code, msg string, err error) error {
	if initJSON {
		return output.Error(code, err.Error())
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// initStandalone writes a full config with defaults and creates its beans
// directory. Returns the beans directory.
func initStandalone(projectDir string, flags *pflag.FlagSet) (string, error) {
	// Config is saved at project root (not inside .beans/), with the
	// directory name as default prefix
	cfg, err := initConfig(config.DefaultWithPrefix(filepath.Base(projectDir)+"-"), flags)
	if err != nil {
		return "", initFail(output.ErrValidation, "invalid flags", err)
	}
	cfg.SetConfigDir(projectDir)

	beansDir := beansPath
	if beansDir == "" {
		beansDir = cfg.ResolveBeansPath()
	}
	if err := beancore.New(beansDir, nil).Init(); err != nil {
		return "", initFail(output.ErrFileError, "failed to create directory", err)
	}

	if err := cfg.Save(projectDir); err != nil {
		return "", initFail(output.ErrFileError, "failed to create config", err)
	}
	// cascade is per file and not part of Config, so it is written separately
	if initCascade {
		if err := config.SetSetting(filepath.Join(projectDir, config.ConfigFileName), "cascade", "true"); err != nil {
			return "", initFail(output.ErrFileError, "failed to set cascade", err)
		}
	}
	return beansDir, nil
}

// initInherited writes a config below a cascading parent that holds only the
// settings given as flags, then creates the beans directory the resulting
// config resolves to (the parent's, unless --path is given). Returns the beans
// directory.
func initInherited(projectDir, parentPath string, flags *pflag.FlagSet) (string, error) {
	parent, err := config.Load(parentPath)
	if err != nil {
		return "", initFail(output.ErrFileError, "failed to load parent config", err)
	}
	// Validate the flags against the parent's statuses and types
	if _, err := initConfig(parent, flags); err != nil {
		return "", initFail(output.ErrValidation, "invalid flags", err)
	}

	settings := initSettings(flags)
	if initCascade {
		settings["cascade"] = true
	}
	if err := config.SaveSettings(projectDir, settings); err != nil {
		return "", initFail(output.ErrFileError, "failed to create config", err)
	}

	cfg, err := config.Load(filepath.Join(projectDir, config.ConfigFileName))
	if err != nil {
		return "", initFail(output.ErrFileError, "failed to load config", err)
	}
	beansDir := cfg.ResolveBeansPath()
	if err := beancore.New(beansDir, nil).Init(); err != nil {
		return "", initFail(output.ErrFileError, "failed to create directory", err)
	}
	return beansDir, nil
}

// initConfig applies the flags the user set to base, validating them against
// base's statuses and types.
func initConfig(base *config.Config, flags *pflag.FlagSet) (*config.Config, error) {
	cfg := base
	beans := &cfg.Beans

	if flags.Changed("path") {
		if initPath == "" {
			return nil, fmt.Errorf("--path must not be empty")
		}
		beans.Path = initPath
	}
	if flags.Changed("prefix") {
		beans.Prefix = initPrefix
	}
	if flags.Changed("id-length") {
		if initIDLength < 1 {
			return nil, fmt.Errorf("--id-length must be at least 1, got %d", initIDLength)
		}
		beans.IDLength = initIDLength
	}
	if flags.Changed("default-status") {
		if !cfg.IsValidStatus(initDefaultStatus) {
			return nil, fmt.Errorf("invalid --default-status %q (must be one of: %s)", initDefaultStatus, cfg.StatusList())
		}
		beans.DefaultStatus = initDefaultStatus
	}
	if flags.Changed("default-type") {
		if !cfg.IsValidType(initDefaultType) {
			return nil, fmt.Errorf("invalid --default-type %q (must be one of: %s)", initDefaultType, cfg.TypeList())
		}
		beans.DefaultType = initDefaultType
	}
	if flags.Changed("prompt-extra") {
		cfg.Prompt = &config.PromptConfig{Extra: initPromptExtra}
	}

	return cfg, nil
}

// initSettingKeys maps each setting flag to its .beans.yml key.
var initSettingKeys = map[string]string{
	"path":           "beans.path",
	"prefix":         "beans.prefix",
	"id-length":      "beans.id_length",
	"default-status": "beans.default_status",
	"default-type":   "beans.default_type",
	"prompt-extra":   "prompt.extra",
}

// initSettings returns only the settings the user passed as flags, keyed by
// their .beans.yml key.
func initSettings(flags *pflag.FlagSet) map[string]any {
	values := map[string]any{
		"path":           initPath,
		"prefix":         initPrefix,
		"id-length":      initIDLength,
		"default-status": initDefaultStatus,
		"default-type":   initDefaultType,
		"prompt-extra":   initPromptExtra,
	}
	settings := map[string]any{}
	for flag, key := range initSettingKeys {
		if flags.Changed(flag) {
			settings[key] = values[flag]
		}
	}
	return settings
}

func init() {
	flags := initCmd.Flags()
	flags.BoolVar(&initJSON, "json", false, "Output as JSON")
	flags.BoolVar(&initInherit, "inherit", false, "Below a cascading parent config: write only the settings given as flags, inherit the rest")
	flags.BoolVar(&initCascade, "cascade", false, "Pass this config's settings down to configs in subdirectories (sets cascade: true)")
	flags.BoolVar(&initForce, "force", false, "Overwrite an existing .beans.yml")
	flags.StringVar(&initPath, "path", "", "Beans directory, relative to the config file (sets beans.path)")
	flags.StringVar(&initPrefix, "prefix", "", "ID prefix for new beans (sets beans.prefix, default: <dir name>-)")
	flags.IntVar(&initIDLength, "id-length", 0, "Length of generated bean IDs (sets beans.id_length, default: 4)")
	flags.StringVar(&initDefaultStatus, "default-status", "", "Status for new beans (sets beans.default_status, default: todo)")
	flags.StringVar(&initDefaultType, "default-type", "", "Type for new beans (sets beans.default_type, default: task)")
	flags.StringVar(&initPromptExtra, "prompt-extra", "", "Text appended to generated bean prompts (sets prompt.extra)")
	rootCmd.AddCommand(initCmd)
}
