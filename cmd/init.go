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
	Long: `Creates a .beans directory and .beans.yml config file in the current directory.

Flags set the matching .beans.yml settings. For example, to keep the beans
outside the project:

  beans init --path ~/path/to/tasks`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fail := func(code, msg string, err error) error {
			if initJSON {
				return output.Error(code, err.Error())
			}
			return fmt.Errorf("%s: %w", msg, err)
		}

		if beansPath != "" && cmd.Flags().Changed("path") {
			return fail(output.ErrValidation, "invalid flags", fmt.Errorf("--path and --beans-path cannot be combined"))
		}

		var projectDir string
		if beansPath != "" {
			// The beans directory is given explicitly; the config goes next to it
			projectDir = filepath.Dir(beansPath)
		} else {
			dir, err := os.Getwd()
			if err != nil {
				return fail(output.ErrFileError, "failed to get current directory", err)
			}
			projectDir = dir
		}

		// Config is saved at project root (not inside .beans/), with the
		// directory name as default prefix
		cfg, err := initConfig(filepath.Base(projectDir), cmd.Flags())
		if err != nil {
			return fail(output.ErrValidation, "invalid flags", err)
		}
		cfg.SetConfigDir(projectDir)

		beansDir := beansPath
		if beansDir == "" {
			beansDir = cfg.ResolveBeansPath()
		}
		if err := beancore.New(beansDir, nil).Init(); err != nil {
			return fail(output.ErrFileError, "failed to create directory", err)
		}

		if err := cfg.Save(projectDir); err != nil {
			return fail(output.ErrFileError, "failed to create config", err)
		}

		if initJSON {
			return output.SuccessInit(beansDir)
		}

		fmt.Println("Initialized beans project")
		return nil
	},
}

// initConfig builds the config for a new project: the defaults, with the
// directory name as prefix, overridden by the flags the user set.
func initConfig(dirName string, flags *pflag.FlagSet) (*config.Config, error) {
	cfg := config.DefaultWithPrefix(dirName + "-")
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

func init() {
	flags := initCmd.Flags()
	flags.BoolVar(&initJSON, "json", false, "Output as JSON")
	flags.StringVar(&initPath, "path", "", "Beans directory, relative to the config file (sets beans.path)")
	flags.StringVar(&initPrefix, "prefix", "", "ID prefix for new beans (sets beans.prefix, default: <dir name>-)")
	flags.IntVar(&initIDLength, "id-length", 0, "Length of generated bean IDs (sets beans.id_length, default: 4)")
	flags.StringVar(&initDefaultStatus, "default-status", "", "Status for new beans (sets beans.default_status, default: todo)")
	flags.StringVar(&initDefaultType, "default-type", "", "Type for new beans (sets beans.default_type, default: task)")
	flags.StringVar(&initPromptExtra, "prompt-extra", "", "Text appended to generated bean prompts (sets prompt.extra)")
	rootCmd.AddCommand(initCmd)
}
